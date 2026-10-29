# Pui

Pui is a keyboard driven terminal note app written in Go. It uses Neovim to edit note text and saves notes and folder metadata locally. Neovim must be installed and available on `PATH`.

## One command install

The installer clones the repository into `~/.pui`, builds the native executable at `~/.pui/target/pui`, and adds `~/.pui/target` to your shell's startup path. It preserves an existing `~/.pui/data` folder. Run:

```sh
curl -fsSL https://raw.githubusercontent.com/pureraku/Pui/main/install.sh | sh -s -- https://github.com/pureraku/Pui.git
```

Or, if you have downloaded `install.sh`, run:

```sh
sh install.sh https://github.com/pureraku/Pui.git
```

You can also set `PUI_REPO_URL` instead of passing the URL. Open a new terminal after installation and run `pui`.

## Manual installation

1. Install [Go](https://go.dev/dl/) and [Neovim](https://neovim.io/).
2. Clone the repository into `~/.pui`:

   ```sh
   git clone https://github.com/pureraku/Pui.git "$HOME/.pui"
   ```

   If `~/.pui` already contains files, use the one-command installer above; it merges the checkout while preserving `data/`.
3. Build the executable into `target/`:

   ```sh
   cd "$HOME/.pui"
   mkdir -p target
   go build -o target/pui ./src
   ```

4. Add the target directory to your `PATH` in `~/.zprofile` (zsh) or `~/.profile` (bash):

   ```sh
   export PATH="$HOME/.pui/target:$PATH"
   ```

5. Open a new terminal and run `pui`.

To build for a different target, set Go's `GOOS` and `GOARCH` environment variables before `go build`; the installer builds for the machine running it.

## Run from source

```sh
cd "$HOME/.pui"
go run ./src
```

## Files and data

The installed layout is:

```text
~/.pui/
├── src/       Go application source
├── target/    built executable (`pui`)
└── data/      notes.json, folders.json, and folders/
```

Pui stores note data in `~/.pui/data/` on macOS, Linux, and other supported systems. On first run after upgrading, it copies existing notes, folders, and folder contents from the previous Pui config folder (on macOS, `~/Library/Application Support/pui/`). The old files are left in place. The migration runs once so deleted notes do not reappear from the old location.

## Keys

- `j`/`k` or arrows — move through the unified folder and note tree
- `g` — jump to Root so you can always create a top-level note or folder
- `enter` — expand/collapse a folder or open a note in Neovim
- `h`/`left` — collapse a folder or move to its parent; `l`/`right` expands
- `n` — create a note in the selected folder (or the selected note's folder)
- `f` — create a nested folder
- `r` — rename the selected folder, including its nested folders and notes
- `d` — delete the selected note or folder; `Enter`/`y` confirms and `n`/`Esc` cancels
- `x` — toggle a note's checked status
- `m` on a note, navigate to a destination folder, then `p` to place it
- `/` — search notes and folders; `Esc` leaves search
- `q` or `ctrl+c` — quit
