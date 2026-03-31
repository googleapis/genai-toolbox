# Compiling for Windows.

Compiling for windows requires the download of zig to provide a C and C++
compiler. These instructions are for cross compiling from Linux x86 but
should work for darwin with small changes.

1. Download zig for your platform.
  ```bash
  cd $HOME
  curl -fL "https://ziglang.org/download/0.15.2/zig-x86_64-linux-0.15.2.tar.xz" -o zig.tar.xz
  tar xf zig.tar.xz
  ```
  This will create the directory $HOME/zig-x86_64-linux-0.15.2. You only need to do this once.

2. Change to your MCP Toolbox directory and run the following:
  ```bash
  cd $HOME/genai-toolbox
  GOOS=windows \
  GOARCH=amd64 \
  CGO_ENABLED=1 \
  CC="$HOME/zig-x86_64-linux-0.15.2/zig cc -target x86_64-windows-gnu"  \
  CXX="$HOME/zig-x86_64-linux-0.15.2/zig c++ -target x86_64-windows-gnu" \
  go build -o toolbox.exe
  ```

Now the toolbox.exe file is ready to use. Transfer it to your windows machine and test it.
