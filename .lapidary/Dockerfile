FROM archlinux:base-devel

# Pacman configuration: parallel downloads
RUN sed -i 's/^#ParallelDownloads/ParallelDownloads/' /etc/pacman.conf

# Install all pacman packages in a single layer and clean cache
RUN pacman-key --init && pacman -Syu --noconfirm && \
  pacman -S --noconfirm --needed \
  # Core/Shell
  zsh \
  tmux \
  sudo \
  less \
  man-db \
  procps-ng \
  which \
  # Version control
  git \
  github-cli \
  # Go
  go \
  delve \
  # Lua
  lua \
  luarocks \
  # Node.js
  nodejs \
  npm \
  # Editors
  neovim \
  # Search/CLI tools
  ripgrep \
  fd \
  fzf \
  jq \
  tree \
  wget \
  curl \
  unzip \
  # Network/Firewall
  iptables \
  ipset \
  iproute2 \
  bind \
  gnupg \
  # Debuggers
  gdb \
  # Build tools (make, cmake + base-devel already in base image)
  make \
  cmake \
  && pacman -Scc --noconfirm

# Create dev user
ARG USERNAME=dev
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME && \
  useradd --uid $USER_UID --gid $USER_GID -m $USERNAME -s /bin/zsh && \
  echo "$USERNAME ALL=(root) NOPASSWD:ALL" > /etc/sudoers.d/$USERNAME && \
  chmod 0440 /etc/sudoers.d/$USERNAME

# Persist command history
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.zsh_history" && \
  mkdir /commandhistory && \
  touch /commandhistory/.zsh_history && \
  chown -R $USERNAME /commandhistory

# Create workspace and config directories
RUN mkdir -p /workspace /home/$USERNAME/.config /home/$USERNAME/.local/share && \
  chown -R $USERNAME:$USERNAME /workspace /home/$USERNAME/.config /home/$USERNAME/.local

# Switch to dev user for user-level setup
USER $USERNAME

# Install LazyVim
RUN git clone https://github.com/LazyVim/starter ~/.config/nvim && \
  rm -rf ~/.config/nvim/.git

# Pre-install LazyVim plugins (headless)
RUN nvim --headless "+Lazy! sync" +qa || true

# Install oh-my-zsh
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended || true

# Switch back to root for system-level setup
USER root

# Install agent CLIs via npm (semicolons so one failure doesn't block the rest)
RUN npm install -g @anthropic-ai/claude-code@latest || true; \
  npm install -g @github/copilot@latest || true; \
  npm install -g @openai/codex@latest || true; \
  npm install -g @google/gemini-cli@latest || true; \
  npm install -g opencode-ai@latest || true; \
  npm install -g @augmentcode/auggie@latest || true; \
  npm install -g @qwen-code/qwen-code@latest || true; \
  npm install -g @tencent-ai/codebuddy-code@latest || true; \
  npm install -g @ast-grep/cli || true; \
  npm cache clean --force

# Install Amazon Q CLI
RUN curl --proto '=https' --tlsv1.2 -sSf \
    'https://desktop-release.q.us-east-1.amazonaws.com/latest/q-x86_64-linux.zip' \
    -o /tmp/q.zip && \
  curl --proto '=https' --tlsv1.2 -sSf \
    'https://desktop-release.q.us-east-1.amazonaws.com/latest/q-x86_64-linux.zip.sig' \
    -o /tmp/q.zip.sig && \
  printf '%s\n' \
    '-----BEGIN PGP PUBLIC KEY BLOCK-----' \
    '' \
    'mDMEZig60RYJKwYBBAHaRw8BAQdAy/+G05U5/EOA72WlcD4WkYn5SInri8pc4Z6D' \
    'BKNNGOm0JEFtYXpvbiBRIENMSSBUZWFtIDxxLWNsaUBhbWF6b24uY29tPoiZBBMW' \
    'CgBBFiEEmvYEF+gnQskUPgPsUNx6jcJMVmcFAmYoOtECGwMFCQPCZwAFCwkIBwIC' \
    'IgIGFQoJCAsCBBYCAwECHgcCF4AACgkQUNx6jcJMVmef5QD/QWWEGG/cOnbDnp68' \
    'SJXuFkwiNwlH2rPw9ZRIQMnfAS0A/0V6ZsGB4kOylBfc7CNfzRFGtovdBBgHqA6P' \
    'zQ/PNscGuDgEZig60RIKKwYBBAGXVQEFAQEHQC4qleONMBCq3+wJwbZSr0vbuRba' \
    'D1xr4wUPn4Avn4AnAwEIB4h+BBgWCgAmFiEEmvYEF+gnQskUPgPsUNx6jcJMVmcF' \
    'AmYoOtECGwwFCQPCZwAACgkQUNx6jcJMVmchMgEA6l3RveCM0YHAGQaSFMkguoAo' \
    'vK6FgOkDawgP0NPIP2oA/jIAO4gsAntuQgMOsPunEdDeji2t+AhV02+DQIsXZpoB' \
    '=f8yY' \
    '-----END PGP PUBLIC KEY BLOCK-----' | gpg --batch --import && \
  gpg --verify /tmp/q.zip.sig /tmp/q.zip && \
  cd /tmp && unzip -q q.zip && \
  chmod +x ./q/install.sh && \
  ./q/install.sh --no-confirm && \
  rm -rf /tmp/q /tmp/q.zip /tmp/q.zip.sig || true

# Copy firewall script
COPY init-firewall.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/init-firewall.sh && \
  echo "$USERNAME ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/$USERNAME-firewall && \
  chmod 0440 /etc/sudoers.d/$USERNAME-firewall

# Switch to dev user
USER $USERNAME

# Environment
ENV SHELL=/bin/zsh
ENV EDITOR=nvim
ENV VISUAL=nvim

WORKDIR /workspace
