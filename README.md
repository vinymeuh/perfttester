# perfttester

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A tool to run Perft tests with my Shogi engines

---

## Installation

```bash
git clone https://codeberg.org/vinymeuh/perfttester.git
cd perfttester && go install .
```
Sub-directories `perfttests` and `unittests` contain tests definition files and are required at runtime.

## Usage

Create `perfttester.yml` or `~/.config/perftester/config.yml` file:

```yaml
engines:
  - name: hifumiz
    path: zig-out/bin/hifumiz

dirtests:
  - name: perfttests
    path: ../perfttester/perfttests
  - name: unittests
    path: ../perfttester/unittests
```

Then:

* run all Perft tests with `perfttester hifumiz`
* run all unittests with `perfttester -d unittests hifumiz`
* run one test in verbose mode `perfttester -t b000.json -v hifumiz`
