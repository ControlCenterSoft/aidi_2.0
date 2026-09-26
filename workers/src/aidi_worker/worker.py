from __future__ import annotations


def health() -> dict[str, str]:
    return {
        "status": "ok",
        "service": "aidi-worker",
        "version": "dev",
    }


if __name__ == "__main__":
    print(health())
