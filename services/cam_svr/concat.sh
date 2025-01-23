#!/bin/bash

# Check if a directory was provided as an argument
if [ "$#" -eq 0 ]; then
    echo "Usage: ./concat.sh <directoryPath>"
    exit 1
fi

# Get the directory path from the first argument
directoryPath=$1

# Check if the directory exists
if [ ! -d "$directoryPath" ]; then
    echo "The specified directory does not exist."
    exit 1
fi

# Create a temporary file to store the list of input files
tempListFile=$(mktemp)

# Find all files with 5 digits in the name (e.g., 00023) and sort them
files=$(find "$directoryPath" -type f -regex ".*/[0-9]\{5\}\.mp4" | sort)

# Write the file list for ffmpeg concat format
for file in $files; do
    echo "file '$file'" >> "$tempListFile"
done

# Define the output file name
outputFilePath="$directoryPath/output.avi"

# Run ffmpeg to concatenate the files
ffmpeg -hide_banner -f concat -safe 0 -i "$tempListFile" -c copy "$outputFilePath"

# Clean up the temporary file
rm -f "$tempListFile"