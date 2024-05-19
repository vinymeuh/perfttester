# perfttester

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

A tool to run Perft tests with my Shogi engines

---

## Build 

```bash
git clone https://codeberg.org/vinymeuh/perfttester.git
cd perfttester && go build .
```
Sub-directories `perfttests` and `unittests` contain tests definition files and are required at runtime.

## How perfttester interacts with an USI engine

Perfttester calls the USI engine with 3 parameters: `/path/to/engine perfttest startpos depth`

A JSON output is expected on stdout and its format depends of the `depth` parameter:

* for `depth=1`

```json
{
  "startpos": "lnsgkgsnl/1r5b1/ppppppppp/9/9/9/PPPPPPPPP/1B5R1/LNSGKGSNL b - 1",
  "moves": [
    "1g1f": 1,
    "1i1h": 1,
    "2g2f": 1,
    "2h1h": 1,
    "2h3h": 1,
    "2h4h": 1,
    "2h5h": 1,
    "2h6h": 1,
    "2h7h": 1,
    "3g3f": 1,
    "3i3h": 1,
    "3i4h": 1,
    "4g4f": 1,
    "4i3h": 1,
    "4i4h": 1,
    "4i5h": 1,
    "5g5f": 1,
    "5i4h": 1,
    "5i5h": 1,
    "5i6h": 1,
    "6g6f": 1,
    "6i5h": 1,
    "6i6h": 1,
    "6i7h": 1,
    "7g7f": 1,
    "7i6h": 1,
    "7i7h": 1,
    "8g8f": 1,
    "9g9f": 1,
    "9i9h": 1
  ] 
}
```

* for `depth>1`

```json
{
  "depth": 2, 
  "nodes": 900
}
```

## Usage

Create `.perfttester.yml` file:

```yaml
engines:
  - name: hifumiz
    path: zig-out/bin/hifumiz

dirtests:
  - name: perft
    path: ../perfttester/testdata/perft
  - name: debug
    path: ../perfttester/testdata/debug
```

Then:

* run all Perft tests with `perfttester hifumiz`
* run all debug tests with `perfttester -d debug hifumiz`
* run one test in verbose mode `perfttester -t b000.json -v hifumiz`
