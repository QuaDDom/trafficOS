import type { Metadata } from 'next';
import './globals.css';
import '@fontsource/barlow-condensed/600.css';
import '@fontsource/barlow-condensed/700.css';
import '@fontsource/ibm-plex-mono/400.css';
import './urban.css';
import './branding.css';
import './control-room.css';
export const metadata: Metadata = { metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000'), title: 'TrafficOS · Urban control system', description: 'Streets alive. Systems connected. A living city where every signal, event and decision is connected.', icons: { icon: [{ url: '/favicon.svg', type: 'image/svg+xml' }, { url: '/brand/app-icon-32.png', type: 'image/png', sizes: '32x32' }], apple: [{ url: '/brand/app-icon-180.png', sizes: '180x180' }] } };
export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) { return <html lang="es"><body>{children}</body></html>; }
