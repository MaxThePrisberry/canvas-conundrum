#!/usr/bin/env python3
"""
Preprocess puzzle images for Canvas Conundrum game.
Splits images into grid segments using letter-row (A, B, C...) and number-column (1, 2, 3...) naming.
"""

import os
import sys
from pathlib import Path
from PIL import Image
import argparse


def center_crop_square(image):
    """
    Center crop an image to make it square using the shorter dimension.
    
    Args:
        image: PIL Image object
    
    Returns:
        PIL Image object (square cropped)
    """
    width, height = image.size
    
    # Determine the size of the square (shorter dimension)
    square_size = min(width, height)
    
    # Calculate cropping box
    left = (width - square_size) // 2
    top = (height - square_size) // 2
    right = left + square_size
    bottom = top + square_size
    
    # Crop and return
    return image.crop((left, top, right, bottom))


def split_image_into_grid(image, grid_size, output_dir, image_id):
    """
    Split an image into a grid of segments.
    
    Args:
        image: PIL Image object (should be square)
        grid_size: Number of rows/columns (e.g., 3 for 3x3, 4 for 4x4)
        output_dir: Directory to save segments
        image_id: Unique identifier for this puzzle image
    """
    # Create output directory if it doesn't exist
    grid_dir = output_dir / f"{grid_size}x{grid_size}"
    grid_dir.mkdir(parents=True, exist_ok=True)
    
    # Get image dimensions
    img_width, img_height = image.size
    
    # Calculate segment dimensions
    segment_width = img_width // grid_size
    segment_height = img_height // grid_size
    
    # Split into segments
    for row in range(grid_size):
        for col in range(grid_size):
            # Calculate segment boundaries
            left = col * segment_width
            top = row * segment_height
            right = left + segment_width
            bottom = top + segment_height
            
            # Handle edge cases for last row/column to include any remaining pixels
            if col == grid_size - 1:
                right = img_width
            if row == grid_size - 1:
                bottom = img_height
            
            # Crop the segment
            segment = image.crop((left, top, right, bottom))
            
            # Generate filename using letter-row, number-column system
            row_letter = chr(65 + row)  # A, B, C, etc.
            col_number = col + 1         # 1, 2, 3, etc.
            filename = f"{row_letter}{col_number}.png"
            
            # Save the segment
            segment_path = grid_dir / filename
            segment.save(segment_path, "PNG", optimize=True)
            
    print(f"  ✓ Created {grid_size}x{grid_size} grid in {grid_dir}")


def process_image(image_path, output_base_dir):
    """
    Process a single image: center crop and split into multiple grid sizes.
    
    Args:
        image_path: Path to the input image
        output_base_dir: Base directory for output
    """
    # Load the image
    try:
        image = Image.open(image_path)
    except Exception as e:
        print(f"Error loading image {image_path}: {e}")
        return False
    
    # Convert to RGB if necessary (handles RGBA, grayscale, etc.)
    if image.mode != 'RGB':
        image = image.convert('RGB')
    
    # Center crop to square
    square_image = center_crop_square(image)
    
    # Get image ID from filename (without extension)
    image_id = Path(image_path).stem
    
    # Create output directory for this image
    image_output_dir = output_base_dir / image_id
    image_output_dir.mkdir(parents=True, exist_ok=True)
    
    # Save the cropped square image for reference
    cropped_path = image_output_dir / "cropped_original.png"
    square_image.save(cropped_path, "PNG", optimize=True)
    print(f"\nProcessing: {image_path}")
    print(f"  Original size: {image.size}")
    print(f"  Cropped size: {square_image.size}")
    print(f"  Saved cropped original to: {cropped_path}")
    
    # Split into different grid sizes (3x3 to 8x8)
    for grid_size in range(3, 9):
        split_image_into_grid(square_image, grid_size, image_output_dir, image_id)
    
    return True


def main():
    parser = argparse.ArgumentParser(
        description="Preprocess images for Canvas Conundrum puzzle game"
    )
    parser.add_argument(
        "image_path",
        type=Path,
        help="Path to the image file to process"
    )
    parser.add_argument(
        "-o", "--output",
        type=Path,
        default=Path("./puzzle_segments"),
        help="Output directory for processed images (default: ./puzzle_segments)"
    )
    
    args = parser.parse_args()
    
    # Validate input
    if not args.image_path.exists():
        print(f"Error: Image file not found: {args.image_path}")
        sys.exit(1)
    
    if not args.image_path.is_file():
        print(f"Error: Path is not a file: {args.image_path}")
        sys.exit(1)
    
    # Process the image
    success = process_image(args.image_path, args.output)
    
    if success:
        print(f"\n✅ Successfully processed {args.image_path}")
        print(f"📁 Output saved to: {args.output / args.image_path.stem}")
    else:
        print(f"\n❌ Failed to process {args.image_path}")
        sys.exit(1)


if __name__ == "__main__":
    main()