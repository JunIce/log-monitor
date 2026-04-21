#!/bin/bash

LOG_DIR="logs"
OUTPUT_FILE="logs/time_ranges.json"

echo "{" > "$OUTPUT_FILE"
echo "  \"generated_at\": \"$(date -Iseconds)\"," >> "$OUTPUT_FILE"
echo "  \"logs\": [" >> "$OUTPUT_FILE"

first=true

for log_file in $(find "$LOG_DIR" -name "*.log" -type f | sort); do
    log_name=$(basename "$log_file")
    dir_name=$(dirname "$log_file" | xargs basename)
    log_type=$(echo "$log_name" | sed 's/\.[0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}\..*\.log$//')
    
    first_time=$(grep -m1 "^[0-9]" "$log_file" | grep -oE "^[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}" | head -n 1)
    last_time=$(grep "^[0-9]" "$log_file" | tail -n 1 | grep -oE "^[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}" | head -n 1)
    
    if [[ -n "$first_time" && -n "$last_time" ]]; then
        if [ "$first" = true ]; then
            first=false
        else
            echo "," >> "$OUTPUT_FILE"
        fi
        echo -n "    {" >> "$OUTPUT_FILE"
        echo -n "\"date\": \"$dir_name\"," >> "$OUTPUT_FILE"
        echo -n "\"filename\": \"$log_name\"," >> "$OUTPUT_FILE"
        echo -n "\"log_type\": \"$log_type\"," >> "$OUTPUT_FILE"
        echo -n "\"first_time\": \"$first_time\"," >> "$OUTPUT_FILE"
        echo -n "\"last_time\": \"$last_time\"" >> "$OUTPUT_FILE"
        echo -n "}" >> "$OUTPUT_FILE"
    fi
done

echo "" >> "$OUTPUT_FILE"
echo "  ]" >> "$OUTPUT_FILE"
echo "}" >> "$OUTPUT_FILE"

echo "Time range info extracted to $OUTPUT_FILE"

cat "$OUTPUT_FILE"