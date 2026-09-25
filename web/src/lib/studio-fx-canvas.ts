/** Particle layers ported from wood-brook-valley-bay.grok.me/studio. */

export function paintStudioFx(
    ctx: CanvasRenderingContext2D,
    width: number,
    height: number,
    time: number,
    fx: string | undefined,
    mode: "light" | "dark",
    nx: number,
    ny: number,
) {
    ctx.clearRect(0, 0, width, height);
    const cx = width * (0.5 + (nx - 0.5) * 0.16);
    const cy = height * (0.42 + (ny - 0.5) * 0.12);
    const alpha = mode === "light" ? 0.28 : 0.55;
    ctx.globalCompositeOperation = mode === "light" ? "multiply" : "lighter";
    if (fx === "rings") paintRings(ctx, width, height, cx, cy, time, alpha);
    else if (fx === "radar") paintRadar(ctx, width, height, cx, cy, time, alpha);
    else if (fx === "rain") paintRain(ctx, width, height, time, alpha);
    else if (fx === "shards") paintShards(ctx, cx, cy, time, alpha);
    else if (fx === "scan") paintScan(ctx, width, height, time, alpha);
    else if (fx === "helix") paintHelix(ctx, cx, cy, time, alpha, height);
    else if (fx === "ink") paintInk(ctx, width, height, time, alpha);
    else if (fx === "sparks") paintSparks(ctx, width, height, time, alpha);
    else if (fx === "aurora") paintAurora(ctx, width, height, time, alpha);
    else paintSonar(ctx, cx, cy, time, alpha);
}

function paintRings(ctx: CanvasRenderingContext2D, width: number, height: number, cx: number, cy: number, time: number, alpha: number) {
    const max = Math.hypot(width, height) * 0.55;
    for (let i = 0; i < 7; i += 1) {
        const radius = (time * 42 + i * 70) % max + 24;
        ctx.beginPath();
        ctx.arc(cx, cy, radius, time * 0.4 + i, time * 0.4 + i + Math.PI * 1.35);
        ctx.strokeStyle = `rgba(217,230,242,${alpha * (1 - radius / max) * 0.7})`;
        ctx.lineWidth = 1.2 + (i % 3) * 0.6;
        ctx.stroke();
    }
    for (let i = 0; i < 48; i += 1) {
        const angle = time * 0.35 + i * 0.131;
        const radius = 40 + (i % 9) * 28 + Math.sin(time + i) * 8;
        ctx.fillStyle = `rgba(255,255,255,${alpha * 0.45})`;
        ctx.fillRect(cx + Math.cos(angle) * radius, cy + Math.sin(angle) * radius * 0.42, 1.4, 1.4);
    }
}

function paintRadar(ctx: CanvasRenderingContext2D, width: number, height: number, cx: number, cy: number, time: number, alpha: number) {
    const sweep = time * 1.15;
    const reach = Math.min(width, height) * 0.42;
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.arc(cx, cy, reach, sweep, sweep + 0.55);
    ctx.closePath();
    ctx.fillStyle = `rgba(94,231,255,${alpha * 0.16})`;
    ctx.fill();
    for (let i = 1; i <= 4; i += 1) {
        ctx.beginPath();
        ctx.arc(cx, cy, i * Math.min(width, height) * 0.09, 0, Math.PI * 2);
        ctx.strokeStyle = `rgba(94,231,255,${alpha * 0.22})`;
        ctx.lineWidth = 1;
        ctx.stroke();
    }
    for (let i = 0; i < 18; i += 1) {
        const angle = time * 0.2 + i * 0.35;
        const radius = 50 + (i % 6) * 36;
        ctx.beginPath();
        ctx.arc(cx + Math.cos(angle) * radius, cy + Math.sin(angle) * radius * 0.7, 2.2, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(180,250,255,${alpha * 0.8})`;
        ctx.fill();
    }
}

function paintRain(ctx: CanvasRenderingContext2D, width: number, height: number, time: number, alpha: number) {
    for (let i = 0; i < 90; i += 1) {
        const x = (i * 97) % width + Math.sin(time + i) * 8;
        const y = (time * (180 + (i % 7) * 40) + i * 73) % (height + 40) - 20;
        ctx.strokeStyle = i % 5 === 0 ? `rgba(180,120,255,${alpha * 0.55})` : `rgba(90,180,255,${alpha * 0.4})`;
        ctx.lineWidth = i % 9 === 0 ? 1.6 : 0.8;
        ctx.beginPath();
        ctx.moveTo(x, y);
        ctx.lineTo(x - 6, y + 22);
        ctx.stroke();
    }
}

function paintShards(ctx: CanvasRenderingContext2D, cx: number, cy: number, time: number, alpha: number) {
    for (let i = 0; i < 14; i += 1) {
        const angle = time * 0.25 + i * ((Math.PI * 2) / 14);
        const radius = 70 + Math.sin(time * 0.8 + i) * 28;
        ctx.save();
        ctx.translate(cx + Math.cos(angle) * radius, cy + Math.sin(angle) * radius);
        ctx.rotate(angle + time * 0.4);
        ctx.beginPath();
        ctx.moveTo(0, -18);
        ctx.lineTo(10, 10);
        ctx.lineTo(-10, 10);
        ctx.closePath();
        ctx.fillStyle = i % 3 === 0 ? `rgba(196,181,255,${alpha * 0.45})` : i % 3 === 1 ? `rgba(120,230,255,${alpha * 0.4})` : `rgba(255,200,140,${alpha * 0.35})`;
        ctx.fill();
        ctx.restore();
    }
}

function paintScan(ctx: CanvasRenderingContext2D, width: number, height: number, time: number, alpha: number) {
    const y = (time * 90) % (height + 80) - 40;
    const gradient = ctx.createLinearGradient(0, y - 30, 0, y + 30);
    gradient.addColorStop(0, "rgba(62,224,176,0)");
    gradient.addColorStop(0.5, `rgba(62,224,176,${alpha * 0.45})`);
    gradient.addColorStop(1, "rgba(62,224,176,0)");
    ctx.fillStyle = gradient;
    ctx.fillRect(0, y - 30, width, 60);
    ctx.strokeStyle = `rgba(180,255,230,${alpha * 0.18})`;
    ctx.lineWidth = 1;
    for (let x = 0; x < width; x += 36) {
        ctx.beginPath();
        ctx.moveTo(x, 0);
        ctx.lineTo(x, height);
        ctx.stroke();
    }
}

function paintHelix(ctx: CanvasRenderingContext2D, cx: number, cy: number, time: number, alpha: number, height: number) {
    for (let strand = 0; strand < 2; strand += 1) {
        ctx.beginPath();
        for (let i = 0; i < 80; i += 1) {
            const t = i / 80;
            const y = cy - height * 0.38 + t * height * 0.76;
            const x = cx + Math.cos(t * 10 + time * 1.4 + strand * Math.PI) * 70;
            if (i === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
        }
        ctx.strokeStyle = strand ? `rgba(110,168,255,${alpha * 0.7})` : `rgba(180,220,255,${alpha * 0.55})`;
        ctx.lineWidth = 1.6;
        ctx.stroke();
    }
    for (let i = 0; i < 24; i += 1) {
        const t = (i / 24 + time * 0.05) % 1;
        const y = cy - height * 0.38 + t * height * 0.76;
        const left = cx + Math.cos(t * 10 + time * 1.4) * 70;
        const right = cx + Math.cos(t * 10 + time * 1.4 + Math.PI) * 70;
        ctx.strokeStyle = `rgba(150,200,255,${alpha * 0.28})`;
        ctx.beginPath();
        ctx.moveTo(left, y);
        ctx.lineTo(right, y);
        ctx.stroke();
    }
}

function paintInk(ctx: CanvasRenderingContext2D, width: number, height: number, time: number, alpha: number) {
    for (let i = 0; i < 6; i += 1) {
        const x = width * (0.2 + (i % 3) * 0.28);
        const y = height * (0.3 + Math.floor(i / 3) * 0.32);
        const radius = 30 + Math.abs(Math.sin(time * 0.5 + i)) * 90;
        ctx.beginPath();
        ctx.arc(x, y, radius, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(20,16,12,${alpha * 0.12})`;
        ctx.fill();
        ctx.strokeStyle = `rgba(232,220,200,${alpha * 0.35})`;
        ctx.lineWidth = 1.2;
        ctx.stroke();
    }
}

function paintSparks(ctx: CanvasRenderingContext2D, width: number, height: number, time: number, alpha: number) {
    for (let i = 0; i < 70; i += 1) {
        const x = (i * 53) % width + Math.sin(time * 2 + i) * 10;
        const y = height - ((time * (40 + (i % 8) * 18) + i * 40) % (height + 20));
        const ember = i % 4 === 0;
        ctx.fillStyle = ember ? `rgba(255,220,140,${alpha * 0.9})` : `rgba(224,138,74,${alpha * 0.7})`;
        ctx.beginPath();
        ctx.arc(x, y, ember ? 2.2 : 1.2, 0, Math.PI * 2);
        ctx.fill();
    }
}

function paintAurora(ctx: CanvasRenderingContext2D, width: number, height: number, time: number, alpha: number) {
    for (let band = 0; band < 3; band += 1) {
        ctx.beginPath();
        for (let x = 0; x <= width; x += 8) {
            const y = height * 0.28 + Math.sin(x * 0.008 + time * 0.6 + band) * 48 + band * 36;
            if (x === 0) ctx.moveTo(x, y);
            else ctx.lineTo(x, y);
        }
        ctx.strokeStyle = band === 1 ? `rgba(160,255,200,${alpha * 0.35})` : `rgba(200,210,230,${alpha * 0.28})`;
        ctx.lineWidth = 10 - band * 2;
        ctx.stroke();
    }
}

function paintSonar(ctx: CanvasRenderingContext2D, cx: number, cy: number, time: number, alpha: number) {
    for (let i = 0; i < 5; i += 1) {
        const radius = (time * 55 + i * 70) % 320 + 10;
        ctx.beginPath();
        ctx.arc(cx, cy, radius, 0, Math.PI * 2);
        ctx.strokeStyle = `rgba(62,200,232,${alpha * (1 - radius / 330) * 0.7})`;
        ctx.lineWidth = 1.4;
        ctx.stroke();
    }
    for (let i = 0; i < 20; i += 1) {
        const angle = time * 0.15 + i * 0.31;
        const radius = 40 + (i % 7) * 22;
        ctx.beginPath();
        ctx.arc(cx + Math.cos(angle) * radius, cy + Math.sin(angle) * radius * 0.55, 1.8, 0, Math.PI * 2);
        ctx.fillStyle = `rgba(160,240,255,${alpha * 0.75})`;
        ctx.fill();
    }
}
