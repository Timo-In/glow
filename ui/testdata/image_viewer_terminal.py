"""Verify LF image rows return to column zero in the actual child terminal."""

import json
import os
from pathlib import Path
import pty
import select
import signal
import sys
import time


root = Path(sys.argv[2])
viewer = root / "viewer"
report = root / "terminal.json"
viewer.write_text(
    "#!" + sys.executable + "\n"
    "import json, os, termios\n"
    "from pathlib import Path\n"
    "flags = termios.tcgetattr(1)[1]\n"
    "Path(os.environ['GLOW_TEST_TERMINAL_REPORT']).write_text(json.dumps({"
    "'opost': bool(flags & termios.OPOST),"
    "'onlcr': bool(flags & termios.ONLCR)}))\n"
    "print('IMAGE_ROW_ONE\\nIMAGE_ROW_TWO', flush=True)\n"
)
viewer.chmod(0o755)

for raw_input, dismiss_key in (("0", b"\r"), ("0", b"\x1b"), ("1", b"\r"), ("1", b"\x1b")):
    pid, master = pty.fork()
    if pid == 0:
        env = os.environ.copy()
        env.update(
            GLOW_TEST_TERMINAL_VIEWER=str(viewer),
            GLOW_TEST_TERMINAL_REPORT=str(report),
            GLOW_TEST_RAW_INPUT=raw_input,
        )
        os.execve(
            sys.argv[1],
            [sys.argv[1], "-test.run=^TestImageViewerTerminalHelper$"],
            env,
        )
    output = b""
    dismissed = False
    reaped = False
    try:
        deadline = time.monotonic() + 8
        while time.monotonic() < deadline:
            if select.select([master], [], [], 0.05)[0]:
                try:
                    output += os.read(master, 65536)
                except OSError:
                    pass
            if not dismissed and b"Press Enter or Esc to return to Glow." in output:
                flags = json.loads(report.read_text())
                assert flags["opost"], "viewer inherited raw output (OPOST disabled)"
                assert flags["onlcr"], "viewer has LF-to-CRLF conversion disabled"
                assert b"IMAGE_ROW_ONE\r\nIMAGE_ROW_TWO\r\n" in output, repr(output)
                # Delayed terminal replies must not dismiss the image.
                os.write(master, b"\x1b]11;rgb:0000/0000/0000\x1b\\")
                time.sleep(0.1)
                done, _ = os.waitpid(pid, os.WNOHANG)
                if done:
                    reaped = True
                    raise AssertionError("terminal reply dismissed image before key press")
                os.write(master, dismiss_key)
                dismissed = True
            done, status = os.waitpid(pid, os.WNOHANG)
            if done:
                reaped = True
                assert dismissed, repr(output)
                assert os.waitstatus_to_exitcode(status) == 0, repr(output)
                break
        else:
            raise AssertionError("image viewer did not return: " + repr(output))
    finally:
        if not reaped:
            try:
                os.kill(pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
            os.waitpid(pid, 0)
        os.close(master)
