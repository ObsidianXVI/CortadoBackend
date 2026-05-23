window.CortadoXterm = {
  _pendingWrites: {},
  _terminals: {},

  init(container, id, onData, onResize) {
    const terminal = new Terminal({
      fontFamily: '"JetBrains Mono", "Fira Code", monospace',
      fontSize: 14,
      cursorBlink: true,
      theme: {
        background: '#101418',
        foreground: '#E7EEF7',
      },
    });
    const fit = new FitAddon.FitAddon();
    const emitResize = () => {
      fit.fit();
      if (typeof onResize === 'function') {
        onResize(terminal.cols, terminal.rows);
      }
    };
    const resizeObserver = new ResizeObserver(emitResize);

    terminal.loadAddon(fit);
    terminal.open(container);
    terminal.onData((data) => {
      if (typeof onData === 'function') {
        onData(data);
      }
    });

    this._terminals[id] = {
      fit,
      resizeObserver,
      terminal,
    };

    resizeObserver.observe(container);
    emitResize();

    const pendingWrites = this._pendingWrites[id];
    if (pendingWrites) {
      pendingWrites.forEach((chunk) => terminal.write(chunk));
      delete this._pendingWrites[id];
    }
  },

  write(id, data) {
    const entry = this._terminals[id];
    if (!entry) {
      if (!this._pendingWrites[id]) {
        this._pendingWrites[id] = [];
      }
      this._pendingWrites[id].push(data);
      return;
    }

    entry.terminal.write(data);
  },

  dispose(id) {
    const entry = this._terminals[id];
    if (!entry) {
      delete this._pendingWrites[id];
      return;
    }

    entry.resizeObserver.disconnect();
    entry.terminal.dispose();
    delete this._pendingWrites[id];
    delete this._terminals[id];
  },
};
