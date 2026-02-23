#!/bin/bash

# Configuration
BUCKET_NAME="mapreduce-wordcount-hw4"
SPLITTER_IP="35.88.54.30"
REDUCER_IP="35.167.132.4"

# Only 3 mapper IPs
MAPPER1_IP="35.93.66.4"
MAPPER2_IP="54.213.105.227"
MAPPER3_IP="44.247.44.108"

# Output file
RESULTS_FILE="benchmark_results.csv"
echo "Experiment,File_Size_MB,Num_Mappers,Total_Time_Seconds,Split_Time,Map_Time,Reduce_Time" > $RESULTS_FILE

run_experiment() {
    local file_size=$1
    local num_mappers=$2
    local input_file="s3://$BUCKET_NAME/inputs/input_${file_size}mb.txt"
    
    echo "=========================================="
    echo "Running: File=${file_size}MB, Mappers=${num_mappers}"
    echo "=========================================="
    
    # Start timing
    start_time=$(date +%s)
    
    # Step 1: Split
    echo "Step 1: Splitting file into $num_mappers chunks..."
    split_start=$(date +%s)
    split_response=$(curl -s -X POST http://$SPLITTER_IP:8080/split \
      -H "Content-Type: application/json" \
      -d "{\"input_url\": \"$input_file\", \"num_chunks\": $num_mappers}")
    split_end=$(date +%s)
    split_time=$((split_end - split_start))
    echo "Split completed in ${split_time}s"
    
    # Extract chunk URLs
    chunk_urls=($(echo $split_response | jq -r '.chunk_urls[]'))
    echo "Created ${#chunk_urls[@]} chunks"
    
    # Step 2: Map (parallel)
    echo "Step 2: Mapping chunks in parallel..."
    map_start=$(date +%s)
    
    # Map chunks based on num_mappers
    if [ $num_mappers -eq 1 ]; then
        map1_response=$(curl -s -X POST http://$MAPPER1_IP:8080/map \
          -H "Content-Type: application/json" \
          -d "{\"chunk_url\": \"${chunk_urls[0]}\"}")
        echo "Mapper 1: $(echo $map1_response | jq -r '.result_url')"
    elif [ $num_mappers -eq 2 ]; then
        curl -s -X POST http://$MAPPER1_IP:8080/map \
          -H "Content-Type: application/json" \
          -d "{\"chunk_url\": \"${chunk_urls[0]}\"}" > /tmp/map1.json &
        curl -s -X POST http://$MAPPER2_IP:8080/map \
          -H "Content-Type: application/json" \
          -d "{\"chunk_url\": \"${chunk_urls[1]}\"}" > /tmp/map2.json &
        wait
        echo "Mapper 1: $(cat /tmp/map1.json | jq -r '.result_url')"
        echo "Mapper 2: $(cat /tmp/map2.json | jq -r '.result_url')"
    else  # 3 mappers
        curl -s -X POST http://$MAPPER1_IP:8080/map \
          -H "Content-Type: application/json" \
          -d "{\"chunk_url\": \"${chunk_urls[0]}\"}" > /tmp/map1.json &
        curl -s -X POST http://$MAPPER2_IP:8080/map \
          -H "Content-Type: application/json" \
          -d "{\"chunk_url\": \"${chunk_urls[1]}\"}" > /tmp/map2.json &
        curl -s -X POST http://$MAPPER3_IP:8080/map \
          -H "Content-Type: application/json" \
          -d "{\"chunk_url\": \"${chunk_urls[2]}\"}" > /tmp/map3.json &
        wait
        echo "Mapper 1: $(cat /tmp/map1.json | jq -r '.result_url')"
        echo "Mapper 2: $(cat /tmp/map2.json | jq -r '.result_url')"
        echo "Mapper 3: $(cat /tmp/map3.json | jq -r '.result_url')"
    fi
    
    map_end=$(date +%s)
    map_time=$((map_end - map_start))
    echo "Map completed in ${map_time}s"
    
    # Construct mapper output URLs (they follow predictable pattern: results/map_chunk_N.json)
    map_result_urls=""
    for ((i=1; i<=num_mappers; i++)); do
        if [ $i -gt 1 ]; then
            map_result_urls="$map_result_urls,"
        fi
        map_result_urls="${map_result_urls}\"s3://$BUCKET_NAME/results/map_chunk_${i}.json\""
    done
    map_urls_json="[$map_result_urls]"
    
    # Step 3: Reduce
    echo "Step 3: Reducing results..."
    reduce_start=$(date +%s)
    reduce_response=$(curl -s -X POST http://$REDUCER_IP:8080/reduce \
      -H "Content-Type: application/json" \
      -d "{\"map_result_urls\": $map_urls_json}")
    reduce_end=$(date +%s)
    reduce_time=$((reduce_end - reduce_start))
    echo "Reduce completed in ${reduce_time}s"
    echo "Final result: $(echo $reduce_response | jq -r '.final_result_url')"
    
    # Calculate total time
    end_time=$(date +%s)
    total_time=$((end_time - start_time))
    
    # Log results
    echo "exp_${file_size}mb_${num_mappers}mappers,$file_size,$num_mappers,$total_time,$split_time,$map_time,$reduce_time" >> $RESULTS_FILE
    
    echo ""
    echo "✓ Experiment completed in ${total_time}s"
    echo "  - Split: ${split_time}s"
    echo "  - Map: ${map_time}s"
    echo "  - Reduce: ${reduce_time}s"
    echo ""
    
    # Clean up for next experiment
    echo "Cleaning up S3..."
    aws s3 rm s3://$BUCKET_NAME/chunks/ --recursive --quiet
    aws s3 rm s3://$BUCKET_NAME/mapped/ --recursive --quiet
    aws s3 rm s3://$BUCKET_NAME/results/ --recursive --quiet
    
    sleep 2
}

echo "======================================"
echo "MapReduce Performance Benchmark"
echo "======================================"
echo "Configuration:"
echo "  Bucket: $BUCKET_NAME"
echo "  Splitter: $SPLITTER_IP"
echo "  Mapper 1: $MAPPER1_IP"
echo "  Mapper 2: $MAPPER2_IP"
echo "  Mapper 3: $MAPPER3_IP"
echo "  Reducer: $REDUCER_IP"
echo "======================================"
echo ""

# Experiment 1: Vary number of mappers with fixed file size (10MB)
echo "=== EXPERIMENT 1: Scaling Mappers (10MB file) ==="
echo "Testing with 1, 2, and 3 mappers"
echo ""
for num_mappers in 1 2 3; do
    run_experiment 10 $num_mappers
done

# Experiment 2: Vary file size with fixed mappers (3 mappers)
echo "=== EXPERIMENT 2: Scaling Data Size (3 mappers) ==="
echo "Testing with 1MB, 5MB, 10MB, 20MB, and 50MB files"
echo ""
for file_size in 1 5 10 20 50; do
    run_experiment $file_size 3
done

echo "======================================"
echo "✓ All benchmarks complete!"
echo "Results saved to: $RESULTS_FILE"
echo "======================================"