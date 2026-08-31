#!/usr/bin/env python3
"""Generate deterministic CC0 synthetic person-centric semantic fixtures."""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageDraw


SIZE = (512, 512)


def person(draw: ImageDraw.ImageDraw, center: tuple[int, int], color: str, scale: float = 1.0) -> None:
    x, y = center
    radius = int(35 * scale)
    draw.ellipse((x - radius, y - 170 * scale, x + radius, y - 100 * scale), fill="#f2c7a5", outline="#33251f", width=4)
    draw.polygon(
        [(x, y - 100 * scale), (x - 80 * scale, y + 90 * scale), (x + 80 * scale, y + 90 * scale)],
        fill=color,
        outline="#222222",
    )
    draw.line((x - 35 * scale, y - 55 * scale, x - 100 * scale, y + 30 * scale), fill="#33251f", width=max(2, int(10 * scale)))
    draw.line((x + 35 * scale, y - 55 * scale, x + 100 * scale, y + 30 * scale), fill="#33251f", width=max(2, int(10 * scale)))


def canvas(color: str) -> tuple[Image.Image, ImageDraw.ImageDraw]:
    image = Image.new("RGB", SIZE, color)
    return image, ImageDraw.Draw(image)


def save_scenes(root: Path) -> None:
    root.mkdir(parents=True, exist_ok=True)

    image, draw = canvas("#171428")
    draw.ellipse((80, 0, 432, 500), fill="#55475f")
    draw.rectangle((0, 405, 512, 512), fill="#2b2038")
    person(draw, (256, 300), "#c82432")
    image.save(root / "red-stage.png")

    image, draw = canvas("#75bfee")
    draw.rectangle((0, 300, 512, 512), fill="#4f9c4b")
    draw.ellipse((380, 30, 470, 120), fill="#ffe36e")
    person(draw, (256, 310), "#276fc4")
    image.save(root / "blue-outdoor.png")

    image, draw = canvas("#241b35")
    draw.polygon([(0, 0), (180, 420), (280, 420)], fill="#7a5c89")
    draw.polygon([(512, 0), (332, 420), (232, 420)], fill="#7a5c89")
    person(draw, (175, 320), "#813ab4", 0.72)
    person(draw, (337, 320), "#e1ad27", 0.72)
    image.save(root / "group-stage.png")

    image, draw = canvas("#e7cfdf")
    draw.ellipse((95, 55, 417, 455), fill="#ff79b5", outline="#542944", width=12)
    draw.ellipse((155, 120, 357, 385), fill="#f2c7a5", outline="#542944", width=7)
    draw.ellipse((205, 220, 230, 245), fill="#2f2731")
    draw.ellipse((282, 220, 307, 245), fill="#2f2731")
    draw.arc((205, 250, 307, 330), 15, 165, fill="#9d3c55", width=6)
    image.save(root / "pink-hair-closeup.png")

    image, draw = canvas("#c8aa88")
    draw.rectangle((65, 55, 447, 465), fill="#f0e2cf", outline="#735d48", width=8)
    draw.rectangle((100, 95, 210, 260), fill="#9bc9e2")
    person(draw, (310, 315), "#f7f5ed")
    image.save(root / "white-indoor.png")

    image, draw = canvas("#071324")
    draw.ellipse((370, 35, 455, 120), fill="#d8e4ff")
    for x, y in ((45, 70), (150, 30), (275, 85), (465, 180)):
        draw.ellipse((x, y, x + 5, y + 5), fill="#ffffff")
    person(draw, (250, 315), "#111111")
    image.save(root / "black-night.png")

    image, draw = canvas("#86c9ef")
    draw.polygon([(0, 330), (150, 120), (270, 330)], fill="#3e8348")
    draw.polygon([(190, 330), (380, 90), (512, 330)], fill="#337342")
    draw.rectangle((0, 330, 512, 512), fill="#65a950")
    image.save(root / "green-landscape.png")

    image, draw = canvas("#f4e4c1")
    draw.polygon([(160, 170), (190, 80), (230, 180)], fill="#d87922")
    draw.polygon([(280, 180), (325, 80), (355, 175)], fill="#d87922")
    draw.ellipse((135, 145, 375, 390), fill="#e58a2b", outline="#563315", width=8)
    draw.ellipse((195, 225, 225, 255), fill="#222222")
    draw.ellipse((285, 225, 315, 255), fill="#222222")
    draw.polygon([(255, 270), (240, 290), (270, 290)], fill="#c05a62")
    image.save(root / "orange-cat.png")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    save_scenes(args.output)


if __name__ == "__main__":
    main()
