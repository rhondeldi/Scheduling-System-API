import os
import re

# Set the directory you want to search in
directory = "./"

# Markdown file to store the TODO list
output_file = "TODO.md"

# Regular expression for matching TODO comments with flexible spacing
todo_pattern = re.compile(r"//\s*TODO\s*:?\s*(.*)")

def write_todo_file(filepath, todos):
    """Write the TODO list to the markdown file."""
    with open(filepath, "w") as md_file:
        md_file.write("# TODO List\n\n")
        count = 1
        for (file, line), todo_text in todos.items():
            md_file.write(f"{count}. File: [{file}]({file}), Line: {line}\n\n")
            md_file.write(f"    {todo_text}\n\n")
            count += 1

# Collect all TODOs from source files
current_todos = {}

# Walk through all files and directories
for root, _, files in os.walk(directory):
    for file in files:
        if file.endswith(".go"):  # Only process .go files
            filepath = os.path.join(root, file)
            try:
                with open(filepath, "r") as f:
                    merged_todo = None  # Store merged TODO line
                    start_line_num = None  # Line number where the TODO starts
                    for line_num, line in enumerate(f, start=1):
                        line = line.rstrip("\n")

                        # Match TODO comments with flexible spacing
                        match = todo_pattern.match(line.strip())
                        if match:
                            if merged_todo:
                                current_todos[(filepath, start_line_num)] = merged_todo
                            merged_todo = match.group(1).strip()
                            start_line_num = line_num

                        # Append continuation lines
                        elif merged_todo and line.strip().startswith("//"):
                            merged_todo += " " + line.strip()[2:].strip()

                        # Write and reset when encountering a non-TODO line
                        else:
                            if merged_todo:
                                current_todos[(filepath, start_line_num)] = merged_todo
                            merged_todo = None

                    # Handle the last TODO block
                    if merged_todo:
                        current_todos[(filepath, start_line_num)] = merged_todo

            except Exception as e:
                print(f"Error reading file {filepath}: {e}")

# Write the collected TODOs to the markdown file
write_todo_file(output_file, current_todos)

print(f"TODO list saved to {output_file}")
