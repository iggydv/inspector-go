#!/usr/bin/env python3
"""Deep compare Python and Go eval logs (unzipped)."""

import json
import difflib
import re
from pathlib import Path


def resolve_log_dir(path: Path) -> Path:
    if not path.exists():
        raise RuntimeError(f"Directory not found: {path}")

    has_json = any(p.suffix == ".json" for p in path.rglob("*.json"))
    if has_json:
        return path

    subdirs = sorted(p for p in path.iterdir() if p.is_dir())
    if not subdirs:
        raise RuntimeError(f"No directories found in {path}")
    return subdirs[-1]


def load_archive(path: Path) -> dict:
    data = {}
    for file_path in sorted(path.rglob("*.json")):
        rel_name = str(file_path.relative_to(path))
        raw = file_path.read_text()
        data[rel_name] = json.loads(raw)
    return data


def normalize_summaries(data: dict) -> dict:
    summaries = []
    summary_key = "_journal/summaries/all.json"
    pattern = re.compile(r"^_journal/summaries/(\d+)\.json$")
    numbered = []
    for key in data:
        match = pattern.match(key)
        if match:
            numbered.append((int(match.group(1)), key))
    if numbered:
        for _, key in sorted(numbered):
            value = data[key]
            if isinstance(value, list):
                summaries.extend(value)
            else:
                summaries.append(value)
        data[summary_key] = summaries
    elif "_journal/summaries/1.json" in data:
        value = data["_journal/summaries/1.json"]
        if isinstance(value, list):
            data[summary_key] = value
        else:
            data[summary_key] = [value]
    return data


def json_lines(obj: object) -> list[str]:
    return json.dumps(obj, indent=2, sort_keys=True).splitlines(keepends=True)


def main() -> None:
    root = Path(__file__).resolve().parents[1]
    py_log = resolve_log_dir(root / "zips" / "py")
    go_log = resolve_log_dir(root / "zips" / "go")

    py_data = normalize_summaries(load_archive(py_log))
    go_data = normalize_summaries(load_archive(go_log))

    py_files = set(py_data.keys())
    go_files = set(go_data.keys())

    report_lines: list[str] = []
    report_lines.append("DEEP LOG COMPARISON\n")
    report_lines.append(f"Python log: {py_log.name}\n")
    report_lines.append(f"Go log:     {go_log.name}\n")
    report_lines.append("\n")

    missing_in_go = sorted(py_files - go_files)
    extra_in_go = sorted(go_files - py_files)
    report_lines.append("FILE LIST DIFFS\n")
    report_lines.append(f"Missing in Go: {missing_in_go if missing_in_go else 'None'}\n")
    report_lines.append(f"Extra in Go:   {extra_in_go if extra_in_go else 'None'}\n")
    report_lines.append("\n")

    common = sorted(py_files & go_files)
    for name in common:
        report_lines.append("=" * 80 + "\n")
        report_lines.append(f"FILE: {name}\n")
        report_lines.append("=" * 80 + "\n")
        py_lines = json_lines(py_data[name])
        go_lines = json_lines(go_data[name])
        diff = list(
            difflib.unified_diff(
                py_lines,
                go_lines,
                fromfile="python",
                tofile="go",
                n=5,
            )
        )
        if diff:
            report_lines.extend(diff)
        else:
            report_lines.append("NO DIFFS\n")
        report_lines.append("\n")

    report_path = root / "log_comparison_report.diff"
    report_path.write_text("".join(report_lines))
    print("✓ Wrote log_comparison_report.diff")
    print(f"Python log: {py_log}")
    print(f"Go log: {go_log}")


if __name__ == "__main__":
    main()
