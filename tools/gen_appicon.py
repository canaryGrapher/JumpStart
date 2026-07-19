"""Generate build/appicon.png: white rocket over teal clouds on a teal
gradient tile, matching the JumpStart brand icon."""
from PIL import Image, ImageDraw
import math

S = 1024
img = Image.new("RGBA", (S, S), (0, 0, 0, 0))
d = ImageDraw.Draw(img)

# Rounded-square tile with vertical teal gradient.
top, bottom = (85, 167, 148), (44, 114, 100)
tile = Image.new("RGBA", (S, S), (0, 0, 0, 0))
td = ImageDraw.Draw(tile)
for y in range(S):
    t = y / S
    col = tuple(int(top[i] + (bottom[i] - top[i]) * t) for i in range(3)) + (255,)
    td.line([(0, y), (S, y)], fill=col)
mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle([16, 16, S - 16, S - 16], radius=230, fill=255)
img.paste(tile, (0, 0), mask)
d = ImageDraw.Draw(img)

def cloud_row(cy, r, color):
    layer = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    ld = ImageDraw.Draw(layer)
    x = -r // 2
    while x < S + r:
        ld.ellipse([x - r, cy - r, x + r, cy + r], fill=color)
        x += int(r * 1.35)
    img.paste(layer, (0, 0), Image.composite(mask, Image.new("L", (S, S), 0),
                                             layer.split()[3]))

cloud_row(1010, 200, (58, 133, 116, 255))   # back clouds
cloud_row(1080, 230, (76, 154, 136, 255))   # front clouds

# Rocket (white), centered, drawn with polygons + ellipse nose.
d = ImageDraw.Draw(img)
white = (244, 247, 246, 255)
shade = (226, 234, 231, 255)
cx = S // 2
body_top, body_bot = 170, 640
w = 150  # half-width

# nose + body as a smooth silhouette
pts = []
steps = 40
for i in range(steps + 1):
    t = i / steps
    y = body_top + (body_bot - body_top) * t
    half = w * math.sin(min(1.0, 0.25 + t * 0.9) * math.pi / 2)
    pts.append((cx - half, y))
for i in range(steps + 1):
    t = 1 - i / steps
    y = body_top + (body_bot - body_top) * t
    half = w * math.sin(min(1.0, 0.25 + t * 0.9) * math.pi / 2)
    pts.append((cx + half, y))
d.polygon(pts, fill=white)
d.ellipse([cx - 60, body_top - 45, cx + 60, body_top + 75], fill=white)

# fins
d.polygon([(cx - w + 8, 480), (cx - w - 110, 660), (cx - w + 20, 645)], fill=shade)
d.polygon([(cx + w - 8, 480), (cx + w + 110, 660), (cx + w - 20, 645)], fill=shade)

# window
d.ellipse([cx - 62, 330, cx + 62, 454], fill=(44, 114, 100, 255))
d.ellipse([cx - 44, 348, cx + 44, 436], fill=(58, 133, 116, 255))

# exhaust
d.polygon([(cx - 62, body_bot - 4), (cx + 62, body_bot - 4), (cx, body_bot + 120)], fill=shade)

img.save("/sessions/amazing-tender-rubin/mnt/jumpstart/build/appicon_new.png")
print("saved")
