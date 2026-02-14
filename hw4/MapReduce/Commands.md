Copy the input file

```
aws s3 cp input.txt s3://mapreduce-wordcount-hw4/input.txt
```

SPLITTER_URL="http://35.92.66.90:8080"
REDUCER_URL="http://16.147.224.68:8080"
MAPPER1="http://34.214.183.77:8080"
MAPPER2="http://34.211.141.33:8080"
MAPPER3="http://34.222.202.72:8080"

curl -s -X POST "$SPLITTER_URL/split" \
 -H "Content-Type: application/json" \
 -d '{"input_url":"s3://mapreduce-wordcount-hw4/input.txt","num_chunks":3}'

# chunk_1 -> mapper1

curl -s -X POST "$MAPPER1/map" -H "Content-Type: application/json" \
 -d '{"chunk_url":"s3://mapreduce-wordcount-hw4/chunks/chunk_1.txt"}'

# chunk_2 -> mapper2

curl -s -X POST "$MAPPER2/map" -H "Content-Type: application/json" \
 -d '{"chunk_url":"s3://mapreduce-wordcount-hw4/chunks/chunk_2.txt"}'

# chunk_3 -> mapper3

curl -s -X POST "$MAPPER3/map" -H "Content-Type: application/json" \
 -d '{"chunk_url":"s3://mapreduce-wordcount-hw4/chunks/chunk_3.txt"}'

curl -s -X POST "$REDUCER_URL/reduce" \
 -H "Content-Type: application/json" \
 -d '{"map_result_urls":["s3://mapreduce-wordcount-hw4/results/map_chunk_1.json","s3://mapreduce-wordcount-hw4/results/map_chunk_2.json","s3://mapreduce-wordcount-hw4/results/map_chunk_3.json"]}'

aws s3 cp s3://mapreduce-wordcount-hw4/results/final_result.json - | jq '.'

Generate test files

python3 generate_test_files.py
