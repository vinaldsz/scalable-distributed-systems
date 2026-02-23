import random
import os

def generate_text_file(filename, size_mb):
    """Generate a text file of approximately size_mb megabytes"""
    words = ['hello', 'world', 'mapreduce', 'distributed', 'systems', 
             'performance', 'testing', 'data', 'processing', 'cloud',
             'computing', 'scalability', 'throughput', 'latency', 'benchmark']
    
    target_size = size_mb * 1024 * 1024  # Convert MB to bytes
    current_size = 0
    
    # Create input directory if it doesn't exist
    os.makedirs(os.path.dirname(filename), exist_ok=True)
    
    with open(filename, 'w') as f:
        while current_size < target_size:
            num_words = random.randint(5, 20)
            sentence = ' '.join(random.choices(words, k=num_words)) + '.\n'
            f.write(sentence)
            current_size += len(sentence.encode('utf-8'))
    
    print(f"Generated {filename}: {current_size / (1024*1024):.2f} MB")

# Generate files of different sizes
if __name__ == "__main__":
    sizes = [1, 5, 10, 20, 50]  # MB
    for size in sizes:
        generate_text_file(f'input/input_{size}mb.txt', size)