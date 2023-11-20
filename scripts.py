from ascii_magic import AsciiArt
import os

directory = "images_input/"

for file in os.listdir(directory):
    my_art = AsciiArt.from_image("images_input/" + file)
    my_art.to_terminal()