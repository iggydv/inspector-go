#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from inspect_ai import Task, task, eval
from inspect_ai.dataset import Sample, hf_dataset
from inspect_ai.scorer import match
from inspect_ai.solver import generate, prompt_template, system_message


def record_to_sample(record: dict) -> Sample:
    delim = "####"
    input_text = record["question"]
    answer = record["answer"].split(delim)
    target = answer.pop().strip()
    reasoning = delim.join(answer)
    return Sample(
        input=input_text,
        target=target,
        metadata={"reasoning": reasoning.strip()},
    )


def sample_to_fewshot(sample: Sample) -> str:
    return (
        f"{sample.input}\n\nReasoning:\n"
        + f"{sample.metadata['reasoning']}\n\n"
        + f"ANSWER: {sample.target}"
    )


MATH_PROMPT_TEMPLATE = """
Solve the following math problem step by step. The last line of your
response should be of the form "ANSWER: $ANSWER" (without quotes)
where $ANSWER is the answer to the problem.

{prompt}

Remember to put your answer on its own line at the end in the form
"ANSWER: $ANSWER" (without quotes) where $ANSWER is the answer to
the problem, and you do not need to use a \\boxed command.

Reasoning:
""".strip()


@task
def gsm8k(fewshot: int = 10, fewshot_seed: int = 42) -> Task:
    solver = [prompt_template(MATH_PROMPT_TEMPLATE), generate()]
    if fewshot > 0:
        fewshots = hf_dataset(
            path="gsm8k",
            data_dir="main",
            split="train",
            sample_fields=record_to_sample,
            shuffle=True,
            seed=fewshot_seed,
            limit=fewshot,
        )
        solver.insert(
            0,
            system_message(
                "\n\n".join(sample_to_fewshot(sample) for sample in fewshots)
            ),
        )

    return Task(
        dataset=hf_dataset(
            path="gsm8k",
            data_dir="main",
            split="test",
            sample_fields=record_to_sample,
        ),
        solver=solver,
        scorer=match(numeric=True),
    )


def main() -> None:
    parser = argparse.ArgumentParser(description="Run GSM8K eval with inspect-ai.")
    parser.add_argument("--model", default="openai/o4-mini")
    parser.add_argument("--fewshot", type=int, default=10)
    parser.add_argument("--fewshot-seed", type=int, default=42)
    parser.add_argument("--log-dir", default="scripts/debug/py_eval_gsm8k")
    args = parser.parse_args()

    root = Path(__file__).resolve().parents[1]
    log_dir = root / args.log_dir
    log_dir.mkdir(parents=True, exist_ok=True)

    eval(
        gsm8k(fewshot=args.fewshot, fewshot_seed=args.fewshot_seed),
        model=args.model,
        log_dir=str(log_dir),
    )


if __name__ == "__main__":
    main()
