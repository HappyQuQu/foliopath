"""Render synthetic test strings only; font is provided read-only, never distributed."""
import argparse
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

parser = argparse.ArgumentParser()
parser.add_argument("font")
parser.add_argument("destination")
args = parser.parse_args()
root = Path(args.destination)
root.mkdir(parents=True, exist_ok=True)
font = ImageFont.truetype(args.font, 52)
image = Image.new("RGB", (1500, 720), "white")
draw = ImageDraw.Draw(image)
for y, text in [(70, "FolioPath OCR 2026"), (210, "中文文字识别测试"), (350, "订单编号 123456"), (490, "LOCAL TEST ONLY")]:
    draw.text((70, y), text, font=font, fill="black")
image.save(root / "bilingual.png")
image.rotate(90, expand=True).save(root / "vertical.png")
Image.new("RGB", (1000, 700), "white").save(root / "blank.png")
Image.new("RGB", (3000, 2000), "white").save(root / "over-pixels.png")
(root / "corrupt.png").write_bytes(b"synthetic invalid image")

wide=Image.new("RGB",(3000,600),"white");ImageDraw.Draw(wide).text((60,120),"WIDE IMAGE OCR 2048",font=font,fill="black");wide.save(root/"wide.png")
exif=Image.Exif();exif[274]=6
image.rotate(90,expand=True).save(root/"exif-oriented.jpg",quality=95,exif=exif)
