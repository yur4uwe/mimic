## Installation / Інструкція встановлення

**ВАЖЛИВО / IMPORTANT:** Для роботи проекту потрібен WebDAV-сервер із Базовою (Basic) автентифікацією. На поточному етапі підтримується лише цей тип авторизації.

Українська
1. Завантажте реліз або повний код з GitHub Releases: https://github.com/yur4uwe/mimic/releases. Для конкретної версії див. https://github.com/yur4uwe/mimic/releases/tag/v0.0.1 (1).
2. Розпакуйте архів і відкрийте папку `build`. Всередині знайдете інсталяційний скрипт для вашої ОС (PowerShell для Windows, Bash для Linux) та інші артефакти.
3. Запустіть відповідний скрипт у середовищі вашої ОС з підвищеними правами (elevated):
	- Windows: відкрийте PowerShell як адміністратор і виконайте `.\install.ps1` або інший скрипт у папці `build`.
	- Linux/macOS: відкрийте термінал, перейдіть у `build` і виконайте `./install.sh` (через `sudo`, якщо потрібно).
	Надайте права адміністратора, якщо скрипт їх запитує — вони потрібні для установки компонентів і копіювання файлів у системні директорії.
4. Введіть параметри, які запитає інсталятор (за потреби). Після завершення команда `mimic` буде доступна у вашому PATH.
5. Перевірте роботу та конфігурацію:
	- `mimic --help` — коротка довідка з параметрами.
	- `mimic --where-config` — шлях до активного конфігураційного файлу у форматі TOML (`*.toml`).
6. ОС-специфічно:
	- Windows: інсталятор може автоматично встановити WinFSP, додати бінар до PATH і помістити конфіг у `%APPDATA%\mimic`.
	- Linux: бінар звичайно встановлюється у `/usr/local/bin`, конфіг — у `/etc/mimic`.

Якщо виникли проблеми — запустіть скрипт з правами адміністратора або відкрийте issue: https://github.com/yur4uwe/mimic/issues

English
1. Download the release archive or the full source from GitHub Releases: https://github.com/yur4uwe/mimic/releases. See the specific tag at https://github.com/yur4uwe/mimic/releases/tag/v0.0.1 (1).
2. Extract the archive and open the `build` directory. There you will find the install scripts and other artifacts for each OS.
3. Run the appropriate install script for your platform with elevated privileges:
	- Windows: open PowerShell as Administrator and run `\\.\\install.ps1` (or the script located in `build`).
	- Linux/macOS: open a terminal, cd into `build`, and run `./install.sh` (use `sudo` if prompted).
	The installer requires elevated permissions to install system components and place files into system directories.
4. Provide any requested information during the install. After the script finishes, the `mimic` command will be available.
5. Useful commands:
	- `mimic --help` — shows usage and flags.
	- `mimic --where-config` — prints the active configuration file location (TOML format, `*.toml`).
6. OS specifics:
	- Windows: the installer may install WinFSP (dependency), add the binary to `PATH`, and place the config in `%APPDATA%\mimic`.
	- Linux: the binary is typically placed in `/usr/local/bin` and the config in `/etc/mimic`.

For problems or questions open an issue: https://github.com/yur4uwe/mimic/issues