# TODO

## 1. Breakpoint visual indicator not rendering
**Symptom:** Console logs `[breakpoint] toggle` and sends `breakpoint_set`/`breakpoint_remove` to backend, but red breakpoint glyph/marker does not appear in Monaco editor gutter.
**Probable causes:**
- `syncBreakpoints` in `EditorView.tsx` depends on `ed.getModel()` which may be `null` during initial render
- Breakpoint decorations created before editor model is ready
- Local vs server breakpoint deduplication logic in `App.tsx` may cause mismatches (local `local-${file}-${line}` vs server IDs)
- `glyphMargin: true` only enabled when `debugState` exists, but breakpoints should show regardless of active debug session

## 2. Hover tooltip on variables not working
**Symptom:** No hover tooltip appears when hovering over PHP variables in editor during debug break.
**Probable causes:**
- `useMonaco()` returns `null` at `handleMount` time (async load), so `registerHoverProvider` never registers
- Hover provider registered only for language `'php'` hardcoded, but debugger is plugin-based and should support any language
- `variables` prop only passed when `debugState.status === 'break'`, but hover needs variables during break
- `detectLanguage()` may not return `'php'` for PHP files
- No fallback for generic variable hover across languages

## 3. Debug panel shows stale state after session ends
**Symptom:** After debug session completes (script finishes), DebugPanel still shows previous variables, stack trace, and STOP button remains lit/red.
**Probable causes:**
- `DebugPanel` renders `state.locals`, `state.stack`, `state.globals` regardless of `state.status`
- No cleanup when `status !== 'break'` — should show empty state
- `useDebugger` hook does not reset `state` to `null` on WebSocket close (`ws.onclose`)
- Backend may not send final `status: 'stopped'`/`'idle'` message, or frontend ignores it

## 4. Debug panel resize handle not working
**Symptom:** Dragging the resize handle between editor and debug panel does not change panel height.
**Probable causes:**
- Formula `window.innerHeight - e.clientY - 24` has wrong sign (dragging up increases height instead of decreasing)
- No `user-select: none` on `body` during drag, causing text selection interference
- Drag listeners on `document` but handle element too thin (4px), mouse may leave element bounds
- Missing `preventDefault()` on `mousedown`/`mousemove`
