#!/usr/bin/env node
// ═══════════════════════════════════════════════════════
//  ⚡ CRYPTO TERMINAL — Live TUI Dashboard
//  Zero dependencies beyond chalk. Pure Node.js.
// ═══════════════════════════════════════════════════════

const chalk = require('chalk');

// ─── Constants ────────────────────────────────────────
const API = 'https://api.coingecko.com/api/v3';
const FG_API = 'https://api.alternative.me/fng/?limit=1';
const REFRESH_MS = 60_000;

// ─── State ────────────────────────────────────────────
let state = {
  coins: [],
  scored: [],
  trending: [],
  global: null,
  fearGreed: null,
  tab: 0,
  selected: 0,
  loading: true,
  lastUpdate: null,
  error: null,
  width: process.stdout.columns || 100,
  height: process.stdout.rows || 30,
};

const TABS = ['📊 Overview', '🔥 Trending', '🔍 Detail', '🌍 Global'];

// ─── API ──────────────────────────────────────────────
async function fetchJSON(url) {
  const res = await fetch(url);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

async function loadAll() {
  state.loading = true;
  render();
  try {
    const [coins, trending, globalData] = await Promise.all([
      fetchJSON(`${API}/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=50&sparkline=true&price_change_percentage=1h,7d,30d`),
      fetchJSON(`${API}/search/trending`),
      fetchJSON(`${API}/global`),
    ]);

    let fg = null;
    try { fg = await fetchJSON(FG_API); } catch {}

    state.coins = coins;
    state.scored = coins.map(computeScore);
    state.trending = trending.coins || [];
    state.global = globalData.data;
    state.fearGreed = fg?.data?.[0] || null;
    state.lastUpdate = new Date();
    state.error = null;
  } catch (e) {
    state.error = e.message;
  }
  state.loading = false;
  render();
}

// ─── Scoring ──────────────────────────────────────────
function computeScore(c) {
  const signals = {};

  // 24h momentum (0-15)
  signals.momentum_24h = Math.round(Math.min(Math.max((c.price_change_percentage_24h || 0) + 15, 0), 30) / 30 * 15 * 10) / 10;

  // 7d momentum (0-20)
  signals.momentum_7d = Math.round(Math.min(Math.max((c.price_change_percentage_7d_in_currency || 0) + 25, 0), 50) / 50 * 20 * 10) / 10;

  // ATH discount (0-20)
  signals.ath_discount = Math.round(Math.min(Math.abs(c.ath_change_percentage || 0) / 80 * 20, 20) * 10) / 10;

  // Volume activity (0-15)
  if (c.market_cap > 0) {
    signals.volume_activity = Math.round(Math.min((c.total_volume / c.market_cap) * 100, 15) * 10) / 10;
  }

  // MCap upside (0-15)
  if (c.market_cap > 1e10) signals.mcap_upside = 3;
  else if (c.market_cap > 1e9) signals.mcap_upside = 6;
  else if (c.market_cap > 5e8) signals.mcap_upside = 9;
  else if (c.market_cap > 1e8) signals.mcap_upside = 12;
  else signals.mcap_upside = 15;

  // Stability (0-15)
  signals.stability = Math.round(Math.max(0, 15 - Math.abs(c.price_change_percentage_1h_in_currency || 0) * 2) * 10) / 10;

  const total = Object.values(signals).reduce((a, b) => a + b, 0);
  return { coin: c, score: Math.round(total * 10) / 10, signals };
}

// ─── Sparkline ────────────────────────────────────────
function sparkline(prices, width = 30) {
  if (!prices || prices.length < 2) return '';
  const step = Math.max(1, Math.floor(prices.length / width));
  const sampled = [];
  for (let i = 0; i < prices.length && sampled.length < width; i += step) sampled.push(prices[i]);

  const min = Math.min(...sampled);
  const max = Math.max(...sampled);
  const blocks = [' ', '⣀', '⣄', '⣆', '⣇', '⣏', '⣟', '⣿'];

  return sampled.map(p => {
    const norm = max > min ? (p - min) / (max - min) : 0.5;
    return blocks[Math.min(Math.floor(norm * (blocks.length - 1)), blocks.length - 1)];
  }).join('');
}

function sparkColor(prices) {
  if (!prices || prices.length < 2) return chalk.dim;
  return prices[prices.length - 1] >= prices[0] ? chalk.green : chalk.red;
}

// ─── Formatters ───────────────────────────────────────
function fmtUSD(v) {
  if (v >= 1e12) return `$${(v / 1e12).toFixed(2)}T`;
  if (v >= 1e9) return `$${(v / 1e9).toFixed(2)}B`;
  if (v >= 1e6) return `$${(v / 1e6).toFixed(2)}M`;
  if (v >= 1e3) return `$${(v / 1e3).toFixed(1)}K`;
  if (v >= 1) return `$${v.toFixed(2)}`;
  return `$${v.toFixed(4)}`;
}

function fmtPct(v) {
  if (v == null) return '—';
  const sign = v >= 0 ? '+' : '';
  return `${sign}${v.toFixed(2)}%`;
}

function pad(s, n) {
  s = String(s);
  return s.length >= n ? s.slice(0, n) : s + ' '.repeat(n - s.length);
}

function colorPct(v) {
  if (v == null) return chalk.dim('—');
  const s = fmtPct(v);
  return v >= 0 ? chalk.green.bold(s) : chalk.red.bold(s);
}

function colorScore(v) {
  if (v >= 70) return chalk.green.bold(String(v));
  if (v >= 50) return chalk.yellow.bold(String(v));
  return chalk.red.bold(String(v));
}

function bar(value, max, width = 20) {
  const filled = Math.round((value / max) * width);
  const b = '█'.repeat(Math.max(0, filled)) + '░'.repeat(Math.max(0, width - filled));
  if (value / max > 0.7) return chalk.green(b);
  if (value / max > 0.4) return chalk.yellow(b);
  return chalk.red(b);
}

// ─── ANSI Helpers ─────────────────────────────────────
const ESC = '\x1b[';
function clear() { process.stdout.write(ESC + '2J' + ESC + 'H'); }
function pos(r, c) { process.stdout.write(ESC + (r + 1) + ';' + (c + 1) + 'H'); }
function hideCursor() { process.stdout.write(ESC + '?25l'); }
function showCursor() { process.stdout.write(ESC + '?25h'); }

// ─── Render ───────────────────────────────────────────
function render() {
  clear();
  const W = state.width;
  const lines = [];

  // Header
  const title = chalk.bold.hex('#9945FF')('⚡ CRYPTO TERMINAL');
  let status = '';
  if (state.loading) status = chalk.dim(' loading...');
  else if (state.lastUpdate) status = chalk.dim(` • updated ${state.lastUpdate.toLocaleTimeString()}`);
  if (state.error) status = chalk.red(` • error: ${state.error}`);
  lines.push(chalk.bgHex('#1a1a2e')(' ' + title + status + ' '.repeat(Math.max(0, W - 40))));

  // Tabs
  const tabs = TABS.map((t, i) => {
    if (i === state.tab) return chalk.bold.hex('#9945FF').underline(t);
    return chalk.dim(t);
  }).join('  ');
  lines.push(' ' + tabs);

  // Content
  switch (state.tab) {
    case 0: lines.push(...renderOverview()); break;
    case 1: lines.push(...renderTrending()); break;
    case 2: lines.push(...renderDetail()); break;
    case 3: lines.push(...renderGlobal()); break;
  }

  // Footer
  lines.push('');
  lines.push(chalk.dim(' tab/→ next • ← prev • ↑↓ nav • enter select • r refresh • 1-4 tabs • q quit'));

  process.stdout.write(lines.join('\n') + '\n');
}

// ─── Overview ─────────────────────────────────────────
function renderOverview() {
  const lines = [''];
  if (!state.scored.length) {
    lines.push(chalk.dim('  Loading market data...'));
    return lines;
  }

  // Header row
  lines.push(
    chalk.dim(
      pad('#', 4) + pad('Coin', 20) + pad('Price', 14) +
      pad('1h', 9) + pad('24h', 9) + pad('7d', 9) +
      pad('MCap', 12) + pad('Score', 7) + '7d Chart'
    )
  );
  lines.push(chalk.dim('─'.repeat(Math.min(state.width - 2, 100))));

  const sparkW = Math.max(10, Math.min(30, state.width - 95));
  const maxRows = Math.max(5, state.height - 12);

  for (let i = 0; i < Math.min(state.scored.length, maxRows); i++) {
    const sc = state.scored[i];
    const c = sc.coin;
    const sel = i === state.selected;

    let name = c.name.length > 16 ? c.name.slice(0, 14) + '..' : c.name;
    let sym = c.symbol.toUpperCase();

    const row =
      chalk.dim(pad(c.market_cap_rank, 4)) +
      chalk.bold(pad(name, 16)) + chalk.dim(pad(sym, 4)) +
      chalk.white(pad(fmtUSD(c.current_price), 14)) +
      colorPct(c.price_change_percentage_1h_in_currency) +
      colorPct(c.price_change_percentage_24h) +
      colorPct(c.price_change_percentage_7d_in_currency) +
      chalk.dim(pad(fmtUSD(c.market_cap), 12)) +
      colorScore(pad(sc.score.toFixed(0), 7)) +
      sparkColor(c.sparkline_in_7d?.price)(sparkline(c.sparkline_in_7d?.price, sparkW));

    lines.push(sel ? chalk.bgHex('#1a1a2e')('▸ ' + row) : '  ' + row);
  }

  return lines;
}

// ─── Trending ─────────────────────────────────────────
function renderTrending() {
  const lines = ['', chalk.bold.cyan('🔥 Top Trending on CoinGecko'), ''];

  if (!state.trending.length) {
    lines.push(chalk.dim('  Loading trending...'));
    return lines;
  }

  state.trending.forEach((tc, i) => {
    const item = tc.item;
    const rank = chalk.green.bold(pad(`#${i + 1}`, 5));
    lines.push(
      `  ${rank} ${chalk.bold(pad(item.name, 25))} ${chalk.dim(item.symbol.toUpperCase())}` +
      (item.market_cap_rank ? chalk.dim(`  MCap Rank: #${item.market_cap_rank}`) : '')
    );
  });

  lines.push('');
  lines.push(chalk.dim('  Data refreshes every 60s • Source: CoinGecko'));
  return lines;
}

// ─── Detail ───────────────────────────────────────────
function renderDetail() {
  const lines = [''];
  if (!state.scored.length) {
    lines.push(chalk.dim('  No data. Press r to refresh.'));
    return lines;
  }

  const sc = state.scored[Math.min(state.selected, state.scored.length - 1)];
  const c = sc.coin;

  lines.push(chalk.bold.cyan(`🔍 ${c.name} (${c.symbol.toUpperCase()}) — Detailed Analysis`));
  lines.push('');

  // Price box
  lines.push(chalk.hex('#2a2a40')('┌' + '─'.repeat(40) + '┐'));
  lines.push(chalk.hex('#2a2a40')('│') + ` Price: ${chalk.bold(fmtUSD(c.current_price))}` + ' '.repeat(Math.max(0, 24 - fmtUSD(c.current_price).length)) + chalk.hex('#2a2a40')('│'));
  lines.push(chalk.hex('#2a2a40')('│') + ` MCap:  ${fmtUSD(c.market_cap)}` + ' '.repeat(Math.max(0, 24 - fmtUSD(c.market_cap).length)) + chalk.hex('#2a2a40')('│'));
  lines.push(chalk.hex('#2a2a40')('│') + ` Vol:   ${fmtUSD(c.total_volume)}` + ' '.repeat(Math.max(0, 24 - fmtUSD(c.total_volume).length)) + chalk.hex('#2a2a40')('│'));
  lines.push(chalk.hex('#2a2a40')('│') + ` Rank:  #${c.market_cap_rank}` + ' '.repeat(Math.max(0, 24 - String(c.market_cap_rank).length - 1)) + chalk.hex('#2a2a40')('│'));
  lines.push(chalk.hex('#2a2a40')('└' + '─'.repeat(40) + '┘'));
  lines.push('');

  // Changes
  lines.push(chalk.bold('  Changes'));
  lines.push(`  1h:  ${colorPct(c.price_change_percentage_1h_in_currency)}`);
  lines.push(`  24h: ${colorPct(c.price_change_percentage_24h)}`);
  lines.push(`  7d:  ${colorPct(c.price_change_percentage_7d_in_currency)}`);
  lines.push(`  30d: ${colorPct(c.price_change_percentage_30d_in_currency)}`);
  lines.push('');

  // ATH
  const athPct = Math.abs(c.ath_change_percentage || 0);
  lines.push(`  ATH: ${chalk.bold(fmtUSD(c.ath))} (${chalk.yellow(athPct.toFixed(1) + '%')} from ATH)`);
  lines.push('');

  // Score breakdown
  lines.push(chalk.bold.cyan('  📊 Score Breakdown'));
  for (const [sig, val] of Object.entries(sc.signals)) {
    const label = pad(sig, 20);
    lines.push(`  ${chalk.dim(label)} ${bar(val, 20)} ${val}`);
  }
  lines.push('');
  lines.push(`  ${chalk.bold(pad('TOTAL SCORE', 20))} ${colorScore(sc.score.toFixed(1))} ${chalk.dim('/ 100')}`);
  lines.push('');

  // Sparkline
  if (c.sparkline_in_7d?.price?.length) {
    lines.push(chalk.bold.cyan('  📈 7-Day Price'));
    lines.push('  ' + sparkColor(c.sparkline_in_7d.price)(sparkline(c.sparkline_in_7d.price, Math.max(20, state.width - 10))));
  }

  lines.push('');
  lines.push(chalk.dim('  ↑/↓ browse • tab switch'));
  return lines;
}

// ─── Global ───────────────────────────────────────────
function renderGlobal() {
  const lines = ['', chalk.bold.cyan('🌍 Global Market Overview'), ''];

  if (!state.global) {
    lines.push(chalk.dim('  Loading global data...'));
    return lines;
  }

  const g = state.global;
  const totalMCap = typeof g.total_market_cap === 'object' ? (g.total_market_cap.usd || 0) : g.total_market_cap;
  const totalVol = typeof g.total_volume === 'object' ? (g.total_volume.usd || 0) : g.total_volume;
  lines.push(chalk.hex('#2a2a40')('┌' + '─'.repeat(50) + '┐'));
  lines.push(chalk.hex('#2a2a40')('│') + ` Total Market Cap: ${chalk.bold(fmtUSD(totalMCap))}`);
  lines.push(chalk.hex('#2a2a40')('│') + ` 24h Change:       ${colorPct(g.market_cap_change_percentage_24h_usd)}`);
  lines.push(chalk.hex('#2a2a40')('│') + ` Active Cryptos:   ${chalk.bold(String(g.active_cryptocurrencies))}`);
  lines.push(chalk.hex('#2a2a40')('│') + ` 24h Volume:       ${chalk.bold(fmtUSD(totalVol))}`);
  lines.push(chalk.hex('#2a2a40')('└' + '─'.repeat(50) + '┘'));
  lines.push('');

  // Fear & Greed
  if (state.fearGreed) {
    const fg = state.fearGreed;
    const fgClass = fg.value_classification;
    let fgColor = chalk.yellow;
    if (/extreme fear/i.test(fgClass)) fgColor = chalk.red.bold;
    else if (/fear/i.test(fgClass)) fgColor = chalk.red;
    else if (/extreme greed/i.test(fgClass)) fgColor = chalk.green.bold;
    else if (/greed/i.test(fgClass)) fgColor = chalk.green;

    lines.push(`  ${chalk.bold('⚡ Fear & Greed:')} ${fgColor(fg.value)} — ${fgColor(fgClass)}`);
    lines.push('');
  }

  // Dominance
  lines.push(chalk.bold.cyan('  📊 Market Dominance'));
  const dom = g.market_cap_percentage || {};
  for (const coin of ['btc', 'eth', 'bnb', 'sol', 'xrp']) {
    if (dom[coin] != null) {
      lines.push(`  ${chalk.bold(pad(coin.toUpperCase(), 5))} ${bar(dom[coin], 65, 25)} ${dom[coin].toFixed(1)}%`);
    }
  }

  return lines;
}

// ─── Input Handling ───────────────────────────────────
function setupInput() {
  if (process.stdin.isTTY) {
    process.stdin.setRawMode(true);
  }
  process.stdin.resume();
  process.stdin.setEncoding('utf8');

  process.stdin.on('data', (key) => {
    // Ctrl+C
    if (key === '\x03') { cleanup(); process.exit(0); }

    switch (key) {
      case 'q': cleanup(); process.exit(0); break;
      case '\t': case 'l': state.tab = (state.tab + 1) % 4; break;
      case 'L': state.tab = (state.tab - 1 + 4) % 4; break;
      case '1': state.tab = 0; break;
      case '2': state.tab = 1; break;
      case '3': state.tab = 2; break;
      case '4': state.tab = 3; break;
      case 'r': loadAll(); return; // don't re-render yet
      case 'j': case '\x1b[B': // down
        if (state.tab === 0 && state.selected < state.scored.length - 1) state.selected++;
        break;
      case 'k': case '\x1b[A': // up
        if (state.tab === 0 && state.selected > 0) state.selected--;
        break;
      case '\r': case '\n': // enter
        if (state.tab === 0) state.tab = 2;
        break;
      case '\x1b[D': // left arrow
        state.tab = (state.tab - 1 + 4) % 4;
        break;
      case '\x1b[C': // right arrow
        state.tab = (state.tab + 1) % 4;
        break;
    }
    render();
  });

  process.stdout.on('resize', () => {
    state.width = process.stdout.columns || 100;
    state.height = process.stdout.rows || 30;
    render();
  });
}

function cleanup() {
  showCursor();
  if (process.stdin.isTTY) {
    process.stdin.setRawMode(false);
  }
  process.stdin.pause();
}

// ─── Main ─────────────────────────────────────────────
async function main() {
  hideCursor();
  setupInput();

  await loadAll();

  // Auto-refresh
  setInterval(() => loadAll(), REFRESH_MS);

  render();
}

main().catch(e => {
  cleanup();
  console.error(e);
  process.exit(1);
});
