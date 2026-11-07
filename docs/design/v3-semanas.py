# Aproxima, por semana ISO y por proyecto, los contadores que deciden las ramas.
# Fuente: ~/.claude/projects/*/*.jsonl (sesiones principales, sin subagentes).
import json, glob, os, re, datetime, collections
RUNNER = re.compile(r'^(?:\w+=\S+\s+)*(?:sudo\s+)?(?:uv run\s+|npx\s+|pnpm\s+|npm\s+(?:run\s+)?)?(pytest|go test|cargo test|vitest|jest|phpunit|tq|make test|just test|php artisan test|pnpm test|npm test|bun test|ctest|rspec)\b')
COMMIT = re.compile(r'^(?:\w+=\S+\s+)*git(?:\s+-\S+(?:\s+\S+)?)*\s+commit\b')
def pieces(cmd): return [p.strip() for p in re.split(r'&&|\|\||;|\|', cmd)]
W = collections.defaultdict(collections.Counter); P = collections.defaultdict(collections.Counter)
for path in glob.glob(os.path.expanduser('~/.claude/projects/*/*.jsonl')):
    proj = os.path.basename(os.path.dirname(path)).replace('-Users-bog-', '').replace('-Users-bog', '~')
    first = None; c = collections.Counter()
    with open(path, errors='ignore') as f:
        for l in f:
            try: o = json.loads(l)
            except Exception: continue
            ts = o.get('timestamp')
            if ts and first is None: first = datetime.datetime.fromisoformat(ts.replace('Z', '+00:00')).astimezone()
            if o.get('subtype') == 'compact_boundary' or o.get('compactMetadata'): c['compacts'] += 1
            if o.get('type') != 'assistant': continue
            for b in (o.get('message', {}).get('content') or []):
                if not isinstance(b, dict) or b.get('type') != 'tool_use': continue
                n, i = b.get('name'), b.get('input') or {}
                if n in ('Edit', 'Write', 'MultiEdit', 'NotebookEdit'): c['edits'] += 1
                elif n == 'Bash':
                    for p in pieces(i.get('command', '')):
                        if COMMIT.match(p): c['commits'] += 1
                        if RUNNER.match(p): c['tests'] += 1
                elif n == 'TodoWrite':
                    c['tasks'] += sum(1 for t in i.get('todos', []) if t.get('status') == 'completed') and 1
                elif n == 'TaskUpdate' and i.get('status') == 'completed': c['tasks'] += 1
    if first is None: continue
    c['sessions'] += 1
    wk = '%d-W%02d' % first.isocalendar()[:2]
    W[wk].update(c); P[proj].update(c)
def temper(c):
    m = (c['commits'] + c['compacts']) / 10; q = (c['tests'] + c['tasks']) / 8
    return ('pauta' if m >= q else 'sonda'), m, q
def trade(c, t):
    if t == 'pauta':
        r, td = c['commits'] / 10, c['sessions'] / 3   # tidy ~ sesiones con poco contexto (cota alta)
        return ('refactor' if r >= td else 'pulcro')
    b, a = c['tests'] / 8, c['tasks'] / 21
    return ('cazabugs' if b >= a else 'arquitecto')
print('SEMANA     ses  com  tst  tar  cmp | metod  inqui -> temperamento / oficio')
for wk in sorted(W):
    c = W[wk]; t, m, q = temper(c)
    print(f"{wk} {c['sessions']:4} {c['commits']:4} {c['tests']:4} {c['tasks']:4} {c['compacts']:4} | {m:5.1f} {q:6.1f} -> {t} / {trade(c, t)}")
print('\nPROYECTO (>=10 sesiones)')
for p, c in sorted(P.items(), key=lambda x: -x[1]['sessions']):
    if c['sessions'] < 10: continue
    t, m, q = temper(c)
    print(f"{p[:34]:34} ses {c['sessions']:3} com {c['commits']:4} tst {c['tests']:4} tar {c['tasks']:4} -> {t} / {trade(c, t)}")
