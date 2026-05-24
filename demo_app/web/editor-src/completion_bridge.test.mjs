import assert from 'node:assert/strict';

import {shouldDropCompletionResult} from './completion_bridge.js';

function entry({cursorVersion, pos}) {
  return {
    cursorVersion,
    view: {
      state: {
        selection: {
          main: {
            head: pos,
          },
        },
      },
    },
  };
}

assert.equal(
  shouldDropCompletionResult(undefined, {
    cursorVersion: 1,
    pos: 12,
  }),
  true,
);

assert.equal(
  shouldDropCompletionResult(
    entry({cursorVersion: 2, pos: 18}),
    {
      cursorVersion: 1,
      pos: 18,
    },
  ),
  true,
);

assert.equal(
  shouldDropCompletionResult(
    entry({cursorVersion: 1, pos: 19}),
    {
      cursorVersion: 1,
      pos: 18,
    },
  ),
  true,
);

assert.equal(
  shouldDropCompletionResult(
    entry({cursorVersion: 3, pos: 24}),
    {
      cursorVersion: 3,
      pos: 24,
    },
  ),
  false,
);
