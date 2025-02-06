#!/bin/bash

echo "Attempting to concatenate files..."

# Check if correct arguments are provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <directoryPath>"
    exit 1
fi

directoryPath="$1"

# Check if the directory exists
if [ ! -d "$directoryPath" ]; then
    echo "Error: The specified directory does not exist."
    exit 1
fi

# Check if ffmpeg is installed
if ! command -v ffmpeg &>/dev/null; then
    echo "Error: ffmpeg is not installed. Please install it and try again."
    exit 1
fi

# Associative arrays for grouping and summing durations
declare -A fileGroups
declare -A durationSums

# Find all .mp4 files, handling spaces properly
while IFS= read -r -d '' file; do
    filename=$(basename "$file")

    # Extract filename sections
    IFS='_' read -r -a sections <<< "$filename"

    # Ensure filename has at least 3 sections
    if [ "${#sections[@]}" -lt 3 ]; then
        echo "Skipping invalid filename: $filename"
        continue
    fi

    section1="${sections[0]}"
    section2="${sections[1]}"
    section3="${sections[2]}"
    sectionLast="${sections[-1]}"  # Last section (usually contains extension)

    # Remove the extension from the last section
    sectionLast="${sectionLast%.mp4}"

    # Ensure section3 is numeric (to avoid sum errors)
    if ! [[ "$section3" =~ ^[0-9]+$ ]]; then
        echo "Warning: Skipping file with non-numeric duration: $filename"
        continue
    fi

    # Create a group key using the first two sections + last section
    groupKey="${section1}_${section2}_${sectionLast}"

    # Store files in an associative array under the respective group key
    fileGroups["$groupKey"]+="$file"$'\n'

    # Sum the duration
    durationSums["$groupKey"]=$((durationSums["$groupKey"] + section3))

done < <(find "$directoryPath" -type f -name "*.mp4" -print0 | sort -z)

# Process each group
for groupKey in "${!fileGroups[@]}"; do
    tempListFile=$(mktemp)
    outputFilePath="$directoryPath/${durationSums[$groupKey]}_${groupKey}.mp4"

    # Sort files based on the second-to-last section (assuming numeric order)
    echo -n "${fileGroups[$groupKey]}" | sort -t '_' -k 5,5n | while IFS= read -r file; do
        if [[ -n "$file" ]]; then
            echo "file '$file'" >> "$tempListFile"
        fi
    done

    # Run ffmpeg to concatenate the files
    if ffmpeg -loglevel error -f concat -safe 0 -i "$tempListFile" -c copy "$outputFilePath"; then
        echo "Created: $outputFilePath"
        
        # Delete input files only if ffmpeg succeeds
        echo -n "${fileGroups[$groupKey]}" | while IFS= read -r file; do
            if [[ -n "$file" ]]; then
                rm -f "$file"
                echo "Deleted: $file"
            fi
        done
    else
        echo "Error: Failed to create $outputFilePath"
    fi

    # Cleanup temporary file
    rm -f "$tempListFile"
done

echo "Concatenation complete."
exit 0
