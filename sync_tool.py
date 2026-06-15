#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import sys
import json
import socket
import struct
import threading
import datetime
import re
import tkinter as tk
from tkinter import ttk, scrolledtext, filedialog

CONFIG_PATH = os.path.join(os.path.expanduser("~"), "wavelog_sync_config.json")

DEFAULTS = {
    "wavelog_url": "",
    "api_key": "",
    "station_id": "",
    "wsjtx_log_path": "",
    "udp_port": 2237,
}

BAND_EDGES = [
    (1800000, 2000000, "160M"),
    (3500000, 4000000, "80M"),
    (5350000, 5450000, "60M"),
    (7000000, 7300000, "40M"),
    (10100000, 10150000, "30M"),
    (14000000, 14350000, "20M"),
    (18068000, 18168000, "17M"),
    (21000000, 21450000, "15M"),
    (24890000, 24990000, "12M"),
    (28000000, 29700000, "10M"),
    (50000000, 54000000, "6M"),
    (144000000, 148000000, "2M"),
    (430000000, 440000000, "70CM"),
]

LOG_QSO_FIELDS = [
    "id", "date", "time", "dx_call", "dx_grid", "tx_freq", "mode",
    "rst_sent", "rst_rcvd", "tx_power", "comments", "name",
    "my_call", "my_grid", "exchange",
]


def default_log_path():
    if sys.platform == "win32":
        return os.path.join(
            os.environ.get("USERPROFILE", "C:/Users/Default"),
            "AppData", "Local", "WSJT-X", "wsjtx_log.adi",
        )
    return os.path.join(
        os.path.expanduser("~"),
        "Library", "Application Support", "WSJT-X", "wsjtx_log.adi",
    )


def freq_to_band(freq_str):
    try:
        f = float(freq_str)
    except (ValueError, TypeError):
        return ""
    for lo, hi, name in BAND_EDGES:
        if lo <= f <= hi:
            return name
    return ""


def adif_field(name, val):
    s = str(val)
    return f"<{name}:{len(s.encode('utf-8'))}>{s}"


def make_adif(qso):
    parts = [adif_field(k, v) for k, v in qso.items() if v]
    parts.append("<EOR>")
    return " ".join(parts)


def extract_adif(text, field):
    m = re.search(rf"<{re.escape(field)}:\d+>(?P<v>[^<]+)", text, re.IGNORECASE)
    return m.group("v").strip() if m else ""


def adif_header():
    now = datetime.datetime.utcnow().strftime("%Y%m%d %H%M%S")
    return (
        f"WSJT-X to Wavelog Sync — exported at {now} UTC\n"
        "<ADIF_VER:5>3.1.0\n"
        "<PROGRAMID:8>wsjt2wavelog\n"
        "<EOH>\n"
    )


def parse_udp_strings(data):
    out, pos = [], 0
    while pos < len(data):
        end = data.find(b"\x00", pos)
        if end == -1:
            break
        out.append(data[pos:end].decode("utf-8", errors="replace"))
        pos = end + 1
        while pos < len(data) and data[pos] == 0:
            pos += 1
    return out


class App:
    def __init__(self, root):
        self.root = root
        root.title("WSJT-X \u2194 Wavelog \u65e5\u5fd7\u540c\u6b65\u5de5\u5177")
        root.geometry("820x680")
        root.minsize(700, 550)

        self.cfg = self._load_cfg()
        self._running = False
        self._sock = None
        self._req = None
        try:
            import requests as r
            self._req = r
        except ImportError:
            pass

        self._build_ui()
        self._apply_cfg()

    # ---- config persistence ----
    def _load_cfg(self):
        try:
            with open(CONFIG_PATH, encoding="utf-8") as f:
                return {**DEFAULTS, **json.load(f)}
        except Exception:
            return dict(DEFAULTS)

    def _save_cfg(self):
        self.cfg["wavelog_url"] = self._url.get().strip()
        self.cfg["api_key"] = self._key.get().strip()
        self.cfg["station_id"] = self._sid.get().strip()
        self.cfg["wsjtx_log_path"] = self._path.get().strip()
        try:
            self.cfg["udp_port"] = int(self._port.get().strip())
        except ValueError:
            self.cfg["udp_port"] = 2237
        try:
            with open(CONFIG_PATH, "w", encoding="utf-8") as f:
                json.dump(self.cfg, f, ensure_ascii=False, indent=2)
            self._log("\u914d\u7f6e\u5df2\u4fdd\u5b58")
        except Exception as e:
            self._log(f"\u4fdd\u5b58\u5931\u8d25: {e}")

    def _apply_cfg(self):
        self._url.set(self.cfg.get("wavelog_url", ""))
        self._key.set(self.cfg.get("api_key", ""))
        self._sid.set(self.cfg.get("station_id", ""))
        p = self.cfg.get("wsjtx_log_path") or default_log_path()
        self._path.set(p)
        self._port.set(str(self.cfg.get("udp_port", 2237)))

    # ---- UI ----
    def _build_ui(self):
        self.root.columnconfigure(0, weight=1)
        self.root.rowconfigure(0, weight=1)
        m = ttk.Frame(self.root, padding=10)
        m.grid(row=0, column=0, sticky="nsew")
        m.columnconfigure(0, weight=1)
        m.rowconfigure(2, weight=1)

        cf = ttk.LabelFrame(m, text="\u914d\u7f6e\u53c2\u6570", padding=8)
        cf.grid(row=0, column=0, sticky="ew", pady=(0, 6))
        cf.columnconfigure(1, weight=1)

        self._url = tk.StringVar()
        self._key = tk.StringVar()
        self._sid = tk.StringVar()
        self._path = tk.StringVar()
        self._port = tk.StringVar()

        r = 0
        ttk.Label(cf, text="Wavelog URL:").grid(row=r, column=0, sticky="w", padx=(0, 4), pady=2)
        ttk.Entry(cf, textvariable=self._url).grid(row=r, column=1, sticky="ew", pady=2)
        r += 1
        ttk.Label(cf, text="API Key:").grid(row=r, column=0, sticky="w", padx=(0, 4), pady=2)
        ef = ttk.Entry(cf, textvariable=self._key, show="*")
        ef.grid(row=r, column=1, sticky="ew", pady=2)
        r += 1
        ttk.Label(cf, text="\u53f0\u7ad9 ID:").grid(row=r, column=0, sticky="w", padx=(0, 4), pady=2)
        ttk.Entry(cf, textvariable=self._sid).grid(row=r, column=1, sticky="ew", pady=2)
        r += 1
        ttk.Label(cf, text="\u672c\u5730\u65e5\u5fd7:").grid(row=r, column=0, sticky="w", padx=(0, 4), pady=2)
        pf = ttk.Frame(cf)
        pf.grid(row=r, column=1, sticky="ew", pady=2)
        ttk.Entry(pf, textvariable=self._path).pack(side="left", fill="x", expand=True)
        ttk.Button(pf, text="\u6d4f\u89c8", width=6, command=self._browse).pack(side="right", padx=(4, 0))
        r += 1
        ttk.Label(cf, text="UDP \u7aef\u53e3:").grid(row=r, column=0, sticky="w", padx=(0, 4), pady=2)
        ttk.Entry(cf, textvariable=self._port, width=10).grid(row=r, column=1, sticky="w", pady=2)
        r += 1
        ttk.Button(cf, text="\u4fdd\u5b58\u914d\u7f6e", command=self._save_cfg).grid(row=r, column=1, sticky="w", pady=4)

        af = ttk.Frame(m)
        af.grid(row=1, column=0, sticky="ew", pady=4)
        self._btn_fwd = ttk.Button(af, text="\u5f00\u542f\u5b9e\u65f6\u8f6c\u53d1", command=self._toggle_fwd)
        self._btn_fwd.pack(side="left", padx=(0, 8))
        self._btn_sync = ttk.Button(af, text="\u7acb\u5373\u540c\u6b65\u8fdc\u7a0b\u65e5\u5fd7", command=self._do_sync)
        self._btn_sync.pack(side="left")

        lf = ttk.LabelFrame(m, text="\u8fd0\u884c\u65e5\u5fd7", padding=4)
        lf.grid(row=2, column=0, sticky="nsew")
        lf.columnconfigure(0, weight=1)
        lf.rowconfigure(0, weight=1)
        self._log_widget = scrolledtext.ScrolledText(
            lf, wrap="word", state="disabled", font=("Consolas", 9),
        )
        self._log_widget.grid(row=0, column=0, sticky="nsew")

    def _browse(self):
        p = filedialog.askopenfilename(
            title="\u9009\u62e9 wsjtx_log.adi",
            filetypes=[("ADIF", "*.adi"), ("All", "*.*")],
        )
        if p:
            self._path.set(p)

    def _log(self, msg):
        self.root.after(0, self._log_impl, msg)

    def _log_impl(self, msg):
        ts = datetime.datetime.now().strftime("%H:%M:%S")
        self._log_widget.configure(state="normal")
        self._log_widget.insert("end", f"[{ts}] {msg}\n")
        self._log_widget.see("end")
        self._log_widget.configure(state="disabled")

    # ---- UDP forward ----
    def _toggle_fwd(self):
        if self._running:
            self._stop_fwd()
        else:
            self._start_fwd()

    def _start_fwd(self):
        if not self._req:
            self._log("\u9519\u8bef: \u8bf7\u5148\u6267\u884c pip install requests")
            return
        self._save_cfg()
        port = self.cfg.get("udp_port", 2237)
        s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        try:
            s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
            s.settimeout(1.0)
            s.bind(("0.0.0.0", port))
        except Exception as e:
            s.close()
            self._log(f"\u65e0\u6cd5\u7ed1\u5b9a UDP {port}: {e}")
            return
        self._sock = s
        self._running = True
        self._btn_fwd.configure(text="\u5173\u95ed\u5b9e\u65f6\u8f6c\u53d1")
        self._log(f"UDP \u76d1\u542c\u5df2\u542f\u52a8 (\u7aef\u53e3 {port})")
        threading.Thread(target=self._udp_loop, daemon=True).start()

    def _stop_fwd(self):
        self._running = False
        self._btn_fwd.configure(text="\u5f00\u542f\u5b9e\u65f6\u8f6c\u53d1")
        if self._sock:
            try:
                self._sock.close()
            except Exception:
                pass
            self._sock = None
        self._log("UDP \u76d1\u542c\u5df2\u505c\u6b62")

    def _udp_loop(self):
        while self._running:
            try:
                d, _ = self._sock.recvfrom(4096)
                self._handle_pkt(d)
            except socket.timeout:
                continue
            except OSError:
                break

    def _handle_pkt(self, data):
        if len(data) < 8:
            return
        mg, typ = struct.unpack_from("<II", data)
        if mg != 0xadbccbda:
            return
        fields = parse_udp_strings(data[8:])
        if typ == 5:
            self._on_log_qso(fields)
        elif typ == 9 and fields:
            self._on_adif(fields[0])

    def _on_log_qso(self, fields):
        m = dict(zip(LOG_QSO_FIELDS, fields))
        date = m.get("date", "").replace("-", "")
        tm = m.get("time", "").replace(":", "")
        call = m.get("dx_call", "")
        if not call:
            return
        band = freq_to_band(m.get("tx_freq", ""))
        mode = m.get("mode", "")
        adif = make_adif({
            "CALL": call,
            "QSO_DATE": date,
            "TIME_ON": tm,
            "BAND": band,
            "MODE": mode,
            "RST_SENT": m.get("rst_sent", ""),
            "RST_RCVD": m.get("rst_rcvd", ""),
            "GRID": m.get("dx_grid", ""),
            "NAME": m.get("name", ""),
            "COMMENT": m.get("comments", ""),
            "QSO_DATE_OFF": date,
            "TIME_OFF": tm,
        })
        self._log(f"\u6536\u5230 QSO: {call}  {band}  {mode}")
        threading.Thread(target=self._fwd_qso, args=(adif,), daemon=True).start()

    def _on_adif(self, raw):
        call = extract_adif(raw, "CALL") or "?"
        self._log(f"\u6536\u5230 ADIF: {call}")
        threading.Thread(target=self._fwd_qso, args=(raw,), daemon=True).start()

    def _fwd_qso(self, adif_str):
        url = self.cfg.get("wavelog_url", "").rstrip("/")
        key = self.cfg.get("api_key", "")
        sid = self.cfg.get("station_id", "")
        if not url or not key:
            self._log("\u672a\u914d\u7f6e Wavelog URL \u6216 API Key")
            return
        try:
            r = self._req.post(
                f"{url}/api/qso",
                json={
                    "key": key,
                    "station_profile_id": sid,
                    "type": "adif",
                    "string": adif_str,
                },
                timeout=15,
            )
            if r.status_code == 200:
                self._log("\u8f6c\u53d1\u6210\u529f")
            else:
                self._log(f"\u8f6c\u53d1\u5931\u8d25 (HTTP {r.status_code}): {r.text[:120]}")
        except self._req.exceptions.Timeout:
            self._log("\u8f6c\u53d1\u8d85\u65f6")
        except self._req.exceptions.ConnectionError:
            self._log("\u65e0\u6cd5\u8fde\u63a5 Wavelog")
        except Exception as e:
            self._log(f"\u8f6c\u53d1\u5f02\u5e38: {e}")

    # ---- pull sync ----
    def _do_sync(self):
        if not self._req:
            self._log("\u9519\u8bef: \u8bf7\u5148\u6267\u884c pip install requests")
            return
        self._save_cfg()
        threading.Thread(target=self._sync_pull, daemon=True).start()

    def _sync_pull(self):
        url = self.cfg.get("wavelog_url", "").rstrip("/")
        key = self.cfg.get("api_key", "")
        sid = self.cfg.get("station_id", "")
        path = self.cfg.get("wsjtx_log_path", "")
        if not url or not key:
            self._log("\u8bf7\u5148\u914d\u7f6e Wavelog URL \u548c API Key")
            return
        if not path:
            self._log("\u8bf7\u5148\u8bbe\u7f6e\u672c\u5730\u65e5\u5fd7\u8def\u5f84")
            return

        try:
            self._log("\u6b63\u5728\u62c9\u53d6\u8fdc\u7a0b\u65e5\u5fd7...")
            r = self._req.post(
                f"{url}/api/get_contacts_adif",
                json={
                    "key": key,
                    "station_profile_id": sid,
                    "output_format": "json",
                    "fields": ["CALL", "BAND", "MODE", "QSO_DATE", "TIME_ON", "RST_RCVD", "RST_SENT"],
                },
                timeout=30,
            )
            if r.status_code != 200:
                self._log(f"\u62c9\u53d6\u5931\u8d25 (HTTP {r.status_code}): {r.text[:120]}")
                return

            data = r.json()
            contacts = data
            if isinstance(data, dict):
                contacts = (data.get("data") or data.get("results") or data.get("contacts") or [])
            if not contacts:
                self._log("\u8fdc\u7a0b\u65e5\u5fd7\u4e3a\u7a7a")
                return
            self._log(f"\u62c9\u53d6\u5230 {len(contacts)} \u6761\u8bb0\u5f55")

            local_keys = set()
            need_head = False
            if os.path.exists(path):
                with open(path, encoding="utf-8") as f:
                    local_keys = self._read_local(f.read())
            else:
                need_head = True

            new_ads = []
            for c in contacts:
                call = str(c.get("CALL", "")).strip().upper()
                qd = str(c.get("QSO_DATE", "")).strip()
                to = str(c.get("TIME_ON", "")).strip()
                if not call or not qd:
                    continue
                if (call, qd, to) in local_keys:
                    continue
                new_ads.append(make_adif({
                    "CALL": call,
                    "QSO_DATE": qd,
                    "TIME_ON": to,
                    "BAND": str(c.get("BAND", "")).strip(),
                    "MODE": str(c.get("MODE", "")).strip(),
                    "RST_SENT": str(c.get("RST_SENT", "")).strip(),
                    "RST_RCVD": str(c.get("RST_RCVD", "")).strip(),
                }))

            if not new_ads:
                self._log("\u672c\u5730\u5df2\u662f\u6700\u65b0")
                return

            with open(path, "a", encoding="utf-8") as f:
                if need_head:
                    f.write(adif_header())
                for a in new_ads:
                    f.write(a + "\n")
            self._log(f"\u65b0\u589e {len(new_ads)} \u6761\u5230 {os.path.basename(path)}")

        except self._req.exceptions.Timeout:
            self._log("\u62c9\u53d6\u8d85\u65f6")
        except self._req.exceptions.ConnectionError:
            self._log("\u65e0\u6cd5\u8fde\u63a5 Wavelog")
        except json.JSONDecodeError:
            self._log("\u8fd4\u56de\u6570\u636e\u4e0d\u662f\u6709\u6548 JSON")
        except Exception as e:
            self._log(f"\u540c\u6b65\u5f02\u5e38: {e}")

    @staticmethod
    def _read_local(content):
        keys = set()
        for rec in re.split(r"<eor>\s*", content, flags=re.IGNORECASE):
            call = extract_adif(rec, "CALL")
            qd = extract_adif(rec, "QSO_DATE")
            to = extract_adif(rec, "TIME_ON")
            if call and qd:
                keys.add((call.upper(), qd.strip(), to.strip()))
        return keys


def main():
    root = tk.Tk()
    App(root)
    root.mainloop()


if __name__ == "__main__":
    main()
