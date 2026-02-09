#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
from pathlib import Path

from datasets import load_dataset


def record_to_sample(record: dict, sample_id: int) -> dict:
    delim = "####"
    answer = record["answer"].split(delim)
    target = answer.pop().strip()
    reasoning = delim.join(answer).strip()
    return {
        "id": str(sample_id),
        "input": record["question"],
        "expected": target,
        "metadata": {"reasoning": reasoning},
    }


def main() -> None:
    parser = argparse.ArgumentParser(description="Export GSM8K to JSONL for Go.")
    parser.add_argument("--split", default="test", choices=["train", "test"])
    parser.add_argument("--config", default="main")
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--shuffle", action="store_true")
    parser.add_argument("--seed", type=int, default=42)
    parser.add_argument("--out", default="datasets/gsm8k_test.jsonl")
    args = parser.parse_args()

    dataset = load_dataset("gsm8k", args.config, split=args.split)
    if args.shuffle:
        dataset = dataset.shuffle(seed=args.seed)
    if args.limit > 0:
        dataset = dataset.select(range(args.limit))

    root = Path(__file__).resolve().parents[1]
    out_path = root / args.out
    out_path.parent.mkdir(parents=True, exist_ok=True)

    with out_path.open("w", encoding="utf-8") as f:
        for idx, record in enumerate(dataset, start=1):
            sample = record_to_sample(record, idx)
            f.write(json.dumps(sample, ensure_ascii=True) + "\n")

    print(f"✓ Wrote {out_path}")


if __name__ == "__main__":
    main()
