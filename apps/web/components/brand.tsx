/** Canonical exported artwork, shared by the app and the downloadable brand kit. */
export function BrandLogo({ className = '', tone = 'light' }: { className?: string; tone?: 'light' | 'dark' }) {
 return <img className={`brand-logo ${className}`} src={`/brand/logo-${tone}.svg`} width={472} height={80} alt="TrafficOS" draggable={false}/>;
}
export function BrandSymbol({ className = '' }: { className?: string }) {
 return <img className={`brand-symbol ${className}`} src="/brand/symbol.svg" width={80} height={80} alt="" aria-hidden="true" draggable={false}/>;
}
