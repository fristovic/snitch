# Homebrew formula for Snitch (macOS only).
# brew install --formula ./packaging/homebrew/snitch.rb

class Snitch < Formula
  desc "Catch Cursor agent lies in prose — local lie detector"
  homepage "https://github.com/fristovic/snitch"
  license "MIT"
  version "0.0.1"

  on_macos do
    on_arm do
      url "https://github.com/fristovic/snitch/releases/download/v0.0.1/snitch_0.0.1_darwin_arm64.tar.gz"
      sha256 "16038d7f8121f9ff8868e9f548f5a9c41998eb731d007c6da515bd9894090734"
    end
    on_intel do
      url "https://github.com/fristovic/snitch/releases/download/v0.0.1/snitch_0.0.1_darwin_amd64.tar.gz"
      sha256 "c879899984cc228d8bd2da3951341dccd7e98a7779a56d2d76b3bb47eaa13a43"
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
