type LogLevel = 'debug' | 'info' | 'warn' | 'error'
type LogFn = (...args: any[]) => void

const LEVELS: Record<LogLevel, number> = { debug: 0, info: 1, warn: 2, error: 3 }

const COLORS: Record<LogLevel, string> = {
  debug: '#7f8c8d',
  info:  '#2980b9',
  warn:  '#e67e22',
  error: '#e74c3c',
}

const getLevelConfig = (): { current: LogLevel; enabled: Set<LogLevel> } => {
  try {
    const stored = localStorage.getItem('app_log_level')
    if (stored && LEVELS[stored as LogLevel] !== undefined) {
      const current = stored as LogLevel
      const enabled = new Set<LogLevel>()
      for (const lvl of Object.keys(LEVELS) as LogLevel[]) {
        if (LEVELS[lvl] >= LEVELS[current]) enabled.add(lvl)
      }
      return { current, enabled }
    }
  } catch { /* localStorage unavailable */ }
  // default: info and above, debug only in dev
  const current = 'info'
  const enabled = new Set<LogLevel>(['info', 'warn', 'error'])
  if (import.meta.env.DEV) enabled.add('debug')
  return { current, enabled }
}

function setLevel(level: LogLevel): void {
  const cfg = getLevelConfig()
  cfg.current = level
  cfg.enabled.clear()
  for (const lvl of Object.keys(LEVELS) as LogLevel[]) {
    if (LEVELS[lvl] >= LEVELS[level]) cfg.enabled.add(lvl)
  }
  // persist
  try { localStorage.setItem('app_log_level', level) } catch { /* ignore */ }
}

function formatMsg(level: LogLevel, args: any[]): any[] {
  const ts = new Date().toISOString().slice(11, 23) // HH:MM:SS.mmm
  const prefix = `[${ts}] [${level.toUpperCase()}]`
  if (typeof args[0] === 'string') {
    return [`${prefix} ${args[0]}`, ...args.slice(1)]
  }
  return [prefix, ...args]
}

function mkLog(level: LogLevel): LogFn {
  return (...args: any[]) => {
    const cfg = getLevelConfig()
    if (!cfg.enabled.has(level)) return
    const msg = formatMsg(level, args)
    const color = COLORS[level]
    switch (level) {
      case 'error': console.error(...msg); break
      case 'warn':  console.warn(...msg); break
      default:
        console.log(`%c${msg[0]}`, `color:${color}`, ...msg.slice(1))
    }
  }
}

export const log = {
  debug: mkLog('debug'),
  info:  mkLog('info'),
  warn:  mkLog('warn'),
  error: mkLog('error'),
  setLevel,
}

export default log
