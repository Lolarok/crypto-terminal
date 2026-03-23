# ⚡ CryptoTerminal — Live TUI Dashboard

Dashboard crypto interattiva nel terminale. Zero browser, zero build step — `node ct.js` e stai guardando dati live.

## 🚀 Come farlo partire

### Prerequisiti
- **Node.js** >= 18 ([download](https://nodejs.org/))
- **Git**

### Passo 1: Clona e installa

```bash
git clone https://github.com/Lolarok/crypto-terminal.git
cd crypto-terminal
npm install
```

### Passo 2: Avvia

```bash
node ct.js
```

Ecco! Dati crypto live nel terminale con sparkline, scoring, e navigazione.

## ⌨️ Controlli

| Tasto | Azione |
|-------|--------|
| `tab` / `→` | Tab successivo |
| `←` | Tab precedente |
| `↑` / `k` | Su |
| `↓` / `j` | Giù |
| `enter` | Seleziona → Detail view |
| `1-4` | Salta a tab specifico |
| `r` | Refresh manuale |
| `q` | Esci |

## 📊 Viste

### 📊 Overview — Top 50 coins
- Prezzo live, variazioni 1h/24h/7d
- Score 0-100 con breakdown multi-fattore
- Sparkline 7 giorni
- Navigazione con ↑↓

### 🔥 Trending
- Top trending su CoinGecko
- Aggiornato ogni 60 secondi

### 🔍 Detail — Analisi profonda
- Prezzo, MCap, Volume, Rank
- Variazioni 1h/24h/7d/30d
- ATH distance
- Score breakdown con barre
- Sparkline 7 giorni

### 🌍 Global
- Total market cap e 24h change
- Fear & Greed Index
- Dominance BTC/ETH/BNB/SOL/XRP

## 📊 Score Model

| Signal | Max | Cosa misura |
|--------|-----|-------------|
| momentum_24h | 15 | Direzione prezzo breve termine |
| momentum_7d | 20 | Trend settimanale |
| ath_discount | 20 | Valore vs ATH (contrarian) |
| volume_activity | 15 | Interesse di trading reale |
| mcap_upside | 15 | Potenziale asimmetrico (small cap) |
| stability | 15 | Stabilità 1h |

**Ratings:** 70+ = 🟢 STRONG | 50-69 = 🟡 WATCH | <50 = 🔴 HOLD

## 📦 Stack

- **Node.js** — runtime
- **chalk** — colori terminal
- **CoinGecko API** — dati crypto (gratuita)
- **Alternative.me** — Fear & Greed (gratuita)
- Zero backend, zero database

## Auto-refresh

I dati si aggiornano automaticamente ogni 60 secondi.
