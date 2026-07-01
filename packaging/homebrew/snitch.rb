# Homebrew formula for Snitch (macOS only).
# brew install --formula ./packaging/homebrew/snitch.rb

class Snitch < Formula
  desc "Catch Cursor agent lies in prose — local lie detector"
  homepage "https://github.com/fristovic/snitch"
  license "MIT"
  version "0.2.1"

  on_macos do
    on_arm do
      url "https://github.com/fristovic/snitch/releases/download/v0.2.1/snitch_0.2.1_darwin_arm64.tar.gz"
      sha256 "5433eabb8c05c4433bf1d6845de7a971a09dbbfaab2b55886a24c0d508c5fc32"
    end
    on_intel do
      url "https://github.com/fristovic/snitch/releases/download/v0.2.1/snitch_0.2.1_darwin_amd64.tar.gz"
      sha256 "b479c3cc14a3c5b0e7af256b7001240a3877e0661aadf2bb1361c64dc18bd5e0"
    end
  end

  def install
    bin.install "snitch"
    bin.install "snitchd"
  end

  def post_install
    return unless OS.mac?

    (var/"log/snitch").mkpath
    plist_path = "#{Dir.home}/Library/LaunchAgents/com.snitch.daemon.plist"
    plist = <<~XML
      <?xml version="1.0" encoding="UTF-8"?>
      <!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
      <plist version="1.0">
      <dict>
        <key>Label</key>
        <string>com.snitch.daemon</string>
        <key>ProgramArguments</key>
        <array><string>#{bin}/snitchd</string></array>
        <key>RunAtLoad</key><true/>
        <key>KeepAlive</key><true/>
        <key>StandardOutPath</key><string>#{Dir.home}/.snitch/snitchd.log</string>
        <key>StandardErrorPath</key><string>#{Dir.home}/.snitch/snitchd.log</string>
      </dict>
      </plist>
    XML
    File.write(plist_path, plist)
    system "launchctl", "bootout", "gui/#{Process.uid}/com.snitch.daemon" rescue nil
    system "launchctl", "bootstrap", "gui/#{Process.uid}", plist_path
  end

  def caveats
    <<~EOS
      Run `snitch status` to verify the daemon.
      Snitch watches Cursor transcripts in ~/.cursor/projects/
    EOS
  end

  test do
    assert_match "Snitch", shell_output("#{bin}/snitch --help")
  end
end
