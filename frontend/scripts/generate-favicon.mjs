// Regenerates public/favicon.ico from the same geometry as public/icon.svg.
// Pure Node (zlib only) — no dependencies, no build step.
//
//   node scripts/generate-favicon.mjs
//
// Emits a multi-resolution .ico (16 / 32 / 48 px) whose frames are PNGs.

import { writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { deflateSync } from "node:zlib";

const GREEN = [0x15, 0x80, 0x3d];
const WHITE = [0xff, 0xff, 0xff];
const SS = 4; // supersampling factor for antialiasing
const SIZES = [16, 32, 48];

// House silhouette pentagon, in the icon.svg 0..64 coordinate space.
const HOUSE = [
  [32, 8],
  [56, 30],
  [56, 56],
  [8, 56],
  [8, 30],
];

function pointInPolygon(x, y, poly) {
  let inside = false;
  for (let i = 0, j = poly.length - 1; i < poly.length; j = i++) {
    const [xi, yi] = poly[i];
    const [xj, yj] = poly[j];
    const hit =
      yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi;
    if (hit) inside = !inside;
  }
  return inside;
}

function inRoundedRect(x, y, size, radius) {
  if (x < 0 || y < 0 || x > size || y > size) return false;
  const cx = Math.min(Math.max(x, radius), size - radius);
  const cy = Math.min(Math.max(y, radius), size - radius);
  const dx = x - cx;
  const dy = y - cy;
  return dx * dx + dy * dy <= radius * radius;
}

function inEuro(x, y) {
  // "C" open to the right: ring around (33.6, 36), clipped left of the opening.
  const dx = x - 33.6;
  const dy = y - 36;
  const d = Math.sqrt(dx * dx + dy * dy);
  if (d >= 10.5 && d <= 15.5 && x < 43) return true;
  // Two horizontal bars.
  if (x >= 17 && x <= 38 && y >= 30.5 && y <= 35.5) return true;
  if (x >= 17 && x <= 38 && y >= 37.5 && y <= 42.5) return true;
  return false;
}

function renderRGBA(size) {
  const radius = (14 / 64) * size;
  const buf = Buffer.alloc(size * size * 4);
  for (let py = 0; py < size; py++) {
    for (let px = 0; px < size; px++) {
      let r = 0;
      let g = 0;
      let b = 0;
      let a = 0;
      for (let sy = 0; sy < SS; sy++) {
        for (let sx = 0; sx < SS; sx++) {
          const x = px + (sx + 0.5) / SS;
          const y = py + (sy + 0.5) / SS;
          if (!inRoundedRect(x, y, size, radius)) continue;
          a += 1;
          const u = (x / size) * 64;
          const v = (y / size) * 64;
          const house = pointInPolygon(u, v, HOUSE) && !inEuro(u, v);
          const [cr, cg, cb] = house ? WHITE : GREEN;
          r += cr;
          g += cg;
          b += cb;
        }
      }
      const n = SS * SS;
      const idx = (py * size + px) * 4;
      if (a === 0) {
        buf[idx] = buf[idx + 1] = buf[idx + 2] = buf[idx + 3] = 0;
      } else {
        buf[idx] = Math.round(r / a);
        buf[idx + 1] = Math.round(g / a);
        buf[idx + 2] = Math.round(b / a);
        buf[idx + 3] = Math.round((a / n) * 255);
      }
    }
  }
  return buf;
}

const CRC_TABLE = (() => {
  const t = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let c = n;
    for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
    t[n] = c >>> 0;
  }
  return t;
})();

function crc32(buf) {
  let c = 0xffffffff;
  for (let i = 0; i < buf.length; i++)
    c = CRC_TABLE[(c ^ buf[i]) & 0xff] ^ (c >>> 8);
  return (c ^ 0xffffffff) >>> 0;
}

function chunk(type, data) {
  const len = Buffer.alloc(4);
  len.writeUInt32BE(data.length, 0);
  const typeBuf = Buffer.from(type, "ascii");
  const crc = Buffer.alloc(4);
  crc.writeUInt32BE(crc32(Buffer.concat([typeBuf, data])), 0);
  return Buffer.concat([len, typeBuf, data, crc]);
}

function encodePNG(size, rgba) {
  const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  const ihdr = Buffer.alloc(13);
  ihdr.writeUInt32BE(size, 0);
  ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8; // bit depth
  ihdr[9] = 6; // color type RGBA
  const raw = Buffer.alloc(size * (size * 4 + 1));
  for (let y = 0; y < size; y++) {
    raw[y * (size * 4 + 1)] = 0; // filter: none
    rgba.copy(raw, y * (size * 4 + 1) + 1, y * size * 4, (y + 1) * size * 4);
  }
  return Buffer.concat([
    sig,
    chunk("IHDR", ihdr),
    chunk("IDAT", deflateSync(raw, { level: 9 })),
    chunk("IEND", Buffer.alloc(0)),
  ]);
}

function encodeICO(frames) {
  const header = Buffer.alloc(6 + 16 * frames.length);
  header.writeUInt16LE(0, 0);
  header.writeUInt16LE(1, 2); // type: icon
  header.writeUInt16LE(frames.length, 4);
  let offset = header.length;
  const bodies = [];
  frames.forEach((f, i) => {
    const e = 6 + i * 16;
    header[e] = f.size >= 256 ? 0 : f.size;
    header[e + 1] = f.size >= 256 ? 0 : f.size;
    header[e + 2] = 0; // palette
    header[e + 3] = 0;
    header.writeUInt16LE(1, e + 4); // planes
    header.writeUInt16LE(32, e + 6); // bpp
    header.writeUInt32LE(f.png.length, e + 8);
    header.writeUInt32LE(offset, e + 12);
    offset += f.png.length;
    bodies.push(f.png);
  });
  return Buffer.concat([header, ...bodies]);
}

const outDir = join(dirname(fileURLToPath(import.meta.url)), "..", "public");
const frames = SIZES.map((size) => ({
  size,
  png: encodePNG(size, renderRGBA(size)),
}));
writeFileSync(join(outDir, "favicon.ico"), encodeICO(frames));
console.log(`wrote public/favicon.ico (${SIZES.join(", ")} px)`);
