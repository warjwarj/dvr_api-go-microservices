#!/bin/bash

# Check if a directory was provided as an argument
if [ "$#" -eq 0 ]; then
    echo "Usage: ./convert.sh <directoryPath>"
    exit 1
fi

# Use the provided directory
directory="$1"

# Check if the directory exists
if [ ! -d "$directory" ]; then
    echo "The specified directory does not exist."
    exit 1
fi

# Change to the provided directory
cd "$directory" || exit

echo "Looking for vid files in: $directory"

# Loop through all .avi files
for avi_file in *; do
    # Check if the file exists
    if [[ -f "$avi_file" ]]; then
        # Define the output mp4 file name (same as input but with .mp4 extension)
        mp4_file="${avi_file}.mp4"
        
        # Convert the .avi file to .mp4 using ffmpeg
        ffmpeg -loglevel quiet -i "$avi_file" -c:v copy -c:a aac "$mp4_file" &
        PID=$!

        # wait for command
        echo "Waiting for ffmpeg conversion ($PID) to finish..."
        wait $PID

        # finish successfully ro not?
        EXIT_STATUS=$!
        
        # Check if the conversion was successful before deleting the .avi file
        if [[ EXIT_STATUS -eq 0 ]]; then
            echo "Conversion successful: $avi_file to $mp4_file"
        else
            echo "Conversion wasn't perfect but probably worked for: $avi_file"
        fi

        # Delete the original .avi file
        rm -f "$avi_file"
    fi
done

exit 0