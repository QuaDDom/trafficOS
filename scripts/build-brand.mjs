// Canonical vector artwork. No font installation is needed to render the logos.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import sharp from 'sharp';
import { zipSync } from 'fflate';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const out = path.join(root, 'apps/web/public/brand');
await fs.mkdir(out, { recursive: true });
const colors = { asphalt: '#1A1D18', signal: '#F17846', chalk: '#E8E5D6', sage: '#B8C69A', concrete: '#858B79' };
// A chamfered road sign with a T junction cut out of its silhouette.
const silhouette = 'M12 0H64V52L52 64H0V12Z M9 16V28H26V55H38V28H55V16Z';
const lane = 'M13 21H21V23H13Z M43 21H51V23H43Z M31 34H33V41H31Z M31 46H33V51H31Z';
const mark = (color = colors.signal, markings = colors.chalk) => `<path fill="${color}" fill-rule="evenodd" d="${silhouette}"/><path fill="${markings}" d="${lane}"/>`;
// Condensed sign lettering: chamfered bowls, open counters, common 10-unit stems.
const glyphs = {
 T: 'M0 0H38V10H24V56H14V10H0Z',
 R: 'M0 0H28L38 10V25L29 34L40 56H28L18 35H10V56H0Z M10 10V25H25L28 22V13L25 10Z',
 A: 'M10 0H29L40 56H29L27 43H12L10 56H0Z M17 12L13 33H25L21 12Z',
 F: 'M0 0H36V10H10V24H30V34H10V56H0Z',
 I: 'M0 0H10V56H0Z',
 C: 'M10 0H38V10H14L10 14V42L14 46H38V56H10L0 46V10Z',
 O: 'M10 0H28L38 10V46L28 56H10L0 46V10Z M14 10L10 14V42L14 46H24L28 42V14L24 10Z',
 S: 'M10 0H38V10H14L10 14V20L14 24H28L38 34V46L28 56H0V46H24L28 42V38L24 34H10L0 24V10Z',
};
function word(color, accent = color) {
 let x = 0;
 return [...'TRAFFICOS'].map((letter, i) => {
  const result = `<path transform="translate(${x} 0)" fill="${i > 6 ? accent : color}" fill-rule="evenodd" d="${glyphs[letter]}"/>`;
  x += (letter === 'I' ? 10 : letter === 'R' || letter === 'A' ? 40 : letter === 'F' ? 36 : 38) + 7;
  return result;
 }).join('');
}
const wordWidth = 370;
const logo = (ink, accent = ink, markColor = colors.signal, markings = ink) => `<g transform="translate(8 8)">${mark(markColor, markings)}</g><g transform="translate(94 12)">${word(ink, accent)}</g>`;
const svg = (width, height, content, title) => `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-label="${title}"><title>${title}</title>${content}</svg>`;
const exports = [];
async function asset(name, width, height, content, pngWidth = width * 4) {
 const source = svg(width, height, content, `TrafficOS — ${name}`);
 await fs.writeFile(path.join(out, `${name}.svg`), source);
 await sharp(Buffer.from(source)).resize(pngWidth).png().toFile(path.join(out, `${name}.png`));
 exports.push({ name, svg: `${name}.svg`, png: `${name}.png`, width: pngWidth, height: Math.round(pngWidth * height / width) });
 return source;
}
await asset('logo-dark', 472, 80, logo(colors.asphalt, colors.asphalt, colors.signal, colors.asphalt));
await asset('logo-light', 472, 80, logo(colors.chalk, colors.signal));
await asset('logo-black', 472, 80, logo('#000000', '#000000', '#000000', '#000000'));
await asset('logo-white', 472, 80, logo('#FFFFFF', '#FFFFFF', '#FFFFFF', '#FFFFFF'));
await asset('wordmark-dark', wordWidth, 56, word(colors.asphalt));
await asset('wordmark-light', wordWidth, 56, word(colors.chalk, colors.signal));
await asset('symbol', 80, 80, `<g transform="translate(8 8)">${mark()}</g>`, 1024);
await asset('symbol-dark', 80, 80, `<g transform="translate(8 8)">${mark(colors.asphalt, colors.asphalt)}</g>`, 1024);
await asset('symbol-white', 80, 80, `<g transform="translate(8 8)">${mark('#FFFFFF', '#FFFFFF')}</g>`, 1024);
await asset('logo-stacked', 400, 220, `<g transform="translate(152 12) scale(1.5)">${mark()}</g><g transform="translate(15 146)">${word(colors.chalk, colors.signal)}</g>`, 1600);
const appIcon = svg(512, 512, `<rect width="512" height="512" fill="${colors.asphalt}"/><g transform="translate(88 88) scale(5.25)">${mark()}</g>`, 'TrafficOS app icon');
await fs.writeFile(path.join(out, 'app-icon.svg'), appIcon);
for (const size of [32, 180, 192, 512]) await sharp(Buffer.from(appIcon)).resize(size).png().toFile(path.join(out, `app-icon-${size}.png`));
const favicon = svg(64, 64, `<rect width="64" height="64" rx="12" fill="${colors.asphalt}"/><g transform="translate(4 4) scale(.875)">${mark()}</g>`, 'TrafficOS');
await fs.writeFile(path.join(root, 'apps/web/public/favicon.svg'), favicon);
await fs.writeFile(path.join(out, 'favicon.svg'), favicon);
const pattern = `<g fill="none" stroke="${colors.signal}" stroke-width="20" opacity=".24">${[0, 1, 2, 3].map(i => `<path d="M${820 + i * 58} -30V${140 + i * 58}L${600 + i * 58} ${360 + i * 58}V690"/>`).join('')}</g>`;
await asset('social-cover', 1200, 630, `<rect width="1200" height="630" fill="${colors.asphalt}"/>${pattern}<path d="M64 60H1136M64 566H1136" stroke="${colors.concrete}" stroke-opacity=".5"/><g transform="translate(58 172) scale(2.05)">${logo(colors.chalk, colors.signal)}</g><text x="76" y="388" fill="${colors.chalk}" font-family="Arial,sans-serif" font-size="32" font-weight="bold">STREETS ALIVE. SYSTEMS CONNECTED.</text><text x="76" y="99" fill="${colors.signal}" font-family="monospace" font-size="14" letter-spacing="3">URBAN MOBILITY LAB</text><text x="76" y="535" fill="${colors.chalk}" font-family="monospace" font-size="14" letter-spacing="2">SIMULATE / OBSERVE / UNDERSTAND</text>`, 1200);
await fs.writeFile(path.join(out, 'palette.json'), JSON.stringify(colors, null, 2) + '\n');
await fs.writeFile(path.join(out, 'assets.json'), JSON.stringify(exports, null, 2) + '\n');
await fs.writeFile(path.join(out, 'README.md'), `# TrafficOS — Brand kit 1.0\n\nUna ciudad viva. Cada decisión, conectada.\n\n## Concepto\nEl símbolo representa una intersección en T recortada en una señal. Sus esquinas y las letras comparten cortes de 45°. Las marcas internas remiten a carriles; el naranja identifica movimiento y acción.\n\n## Archivos\n- logo-light: horizontal para asfalto y fondos oscuros.\n- logo-dark: horizontal para blanco o papel claro.\n- logo-black / logo-white: una sola tinta, fondos contrastantes.\n- symbol: avatar y símbolo aislado en fondo oscuro. symbol-dark / symbol-white: una tinta.\n- wordmark: lettering sin símbolo; stacked: composición vertical en fondo oscuro.\n- app-icon: iconos con fondo, 32/180/192/512 px.\n- social-cover: portada 1200 × 630 px.\n- Los PNG de logos tienen fondo transparente. Los SVG de logos contienen trazados, no texto ni fuentes externas. La portada usa texto de sistema.\n\n## Uso\nMantener al menos 16 unidades libres alrededor del lockup horizontal de 472 × 80. Ancho recomendado mínimo: 142 px; símbolo aislado: 24 px (favicon simplificado para tamaños menores). No deformar, rotar, agregar sombras, cambiar colores o colocar sobre imágenes sin una superficie uniforme. Usar SVG en web e impresión vectorial y PNG en herramientas que lo requieran.\n\n## Colores\n${Object.entries(colors).map(([name, value]) => '- ' + name + ': ' + value).join('\n')}\n\nEl naranja es acento de marca. El estado de semáforos siempre incluye texto o forma, además del color.\n\n## Tipografía y voz\nEl lettering del logo es dibujo vectorial propio. Barlow Condensed 600/700 para titulares; IBM Plex Mono 400 para datos; Arial/sans-serif para lectura extensa. Las fuentes de interfaz se distribuyen en sus paquetes @fontsource con sus licencias respectivas; no forman parte de estos trazados.\nVoz: directa, urbana y comprensible. Frases cortas, verbos concretos y decisiones explicables. Tagline: Streets alive. Systems connected.\n\n## Fuente editable\nGenerar desde scripts/build-brand.mjs con npm run brand:build. Identidad original dibujada para este proyecto.\n`);
console.log(`Generated ${exports.length} SVG/PNG pairs plus app icons in ${out}`);
await fs.copyFile(path.join(out, 'social-cover.png'), path.join(root, 'apps/web/app/opengraph-image.png'));
const archive = {};
for (const file of (await fs.readdir(out)).filter(name => !name.endsWith('.zip')).sort()) {
 archive[`trafficos-brand-kit/${file}`] = new Uint8Array(await fs.readFile(path.join(out, file)));
}
await fs.writeFile(path.join(out, 'trafficos-brand-kit.zip'), zipSync(archive, { level: 6 }));
