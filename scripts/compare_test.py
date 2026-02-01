#!/usr/bin/env python3
from pathlib import Path
import shutil
import zipfile

from inspect_ai import Task, task, eval
from inspect_ai.dataset import FieldSpec, json_dataset
from inspect_ai.scorer import exact
from inspect_ai.solver import generate, system_message


@task
def simple_math():
    dataset = json_dataset(
        "examples/math/dataset.jsonl",
        FieldSpec(input="input", target="expected", id="id"),
    )
    return Task(
        dataset=dataset,
        solver=[
            system_message("Solve the problem and return only the final answer with no extra text."),
            generate(),
        ],
        scorer=exact(),
    )


def clear_directory(path: Path) -> None:
    if not path.exists():
        return
    for child in path.iterdir():
        if child.is_dir():
            shutil.rmtree(child)
        else:
            child.unlink()


if __name__ == "__main__":
    root = Path(__file__).resolve().parents[1]
    log_dir = root / "scripts" / "debug" / "py_eval"
    extract_dir = root / "scripts" / "debug" / "py"

    log_dir.mkdir(parents=True, exist_ok=True)
    extract_dir.mkdir(parents=True, exist_ok=True)

    log = eval(simple_math(), model="openai/o4-mini", log_dir=str(log_dir))[0]
    log_path = Path(log.location)

    clear_directory(extract_dir)
    with zipfile.ZipFile(log_path, "r") as zf:
        zf.extractall(extract_dir)

    print(f"\n✓ Python log created: {log.location}")
    print(f"✓ Extracted to: {extract_dir}")
