#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  scripts/capture_fixture.sh --agent AGENT --status STATUS --pane PANE --name NAME [options]

Options:
  --agent AGENT        Fixture agent directory, for example: claude, codex, opencode
  --status STATUS      Fixture status directory, for example: working, waiting, idle, error, unknown
  --pane PANE          tmux pane target, for example: %3
  --name NAME          Fixture file basename, without .txt
  --lines N            Capture the last N lines from the pane (default: 80)
  --capture-title      Also save the current pane title to a sibling .title.txt file
  --root PATH          Fixture root (default: <repo>/testdata/fixtures)
  -h, --help           Show this help text

Examples:
  scripts/capture_fixture.sh --agent codex --status working --pane %3 --name current_working_status_bar
  scripts/capture_fixture.sh --agent claude --status waiting --pane %5 --name permission_prompt --capture-title
EOF
}

agent=""
status=""
pane=""
name=""
lines="80"
capture_title="false"
script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
fixture_root="${repo_root}/testdata/fixtures"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --agent)
      agent="${2:-}"
      shift 2
      ;;
    --status)
      status="${2:-}"
      shift 2
      ;;
    --pane)
      pane="${2:-}"
      shift 2
      ;;
    --name)
      name="${2:-}"
      shift 2
      ;;
    --lines)
      lines="${2:-}"
      shift 2
      ;;
    --capture-title)
      capture_title="true"
      shift
      ;;
    --root)
      fixture_root="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ -z "${agent}" || -z "${status}" || -z "${pane}" || -z "${name}" ]]; then
  echo "Missing required arguments." >&2
  usage >&2
  exit 1
fi

if ! [[ "${lines}" =~ ^[0-9]+$ ]]; then
  echo "--lines must be a positive integer." >&2
  exit 1
fi

dest_dir="${fixture_root}/${agent}/${status}"
dest_file="${dest_dir}/${name}.txt"

mkdir -p "${dest_dir}"
tmux capture-pane -p -t "${pane}" -S "-${lines}" > "${dest_file}"
echo "Saved pane capture to ${dest_file}"

if [[ "${capture_title}" == "true" ]]; then
  title_file="${dest_dir}/${name}.title.txt"
  tmux display-message -p -t "${pane}" '#{pane_title}' > "${title_file}"
  echo "Saved pane title to ${title_file}"
fi
