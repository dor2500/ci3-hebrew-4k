# Renders the Hebrew voice lines with the Windows "Microsoft Asaf" voice.
# Run once from the repo root; needs ffmpeg. The OGG files are embedded into the exe.
Add-Type -AssemblyName System.Speech
$lines = @{
  yay = 'יאיי!'; kol = 'כל הכבוד!'; metzuyan = 'מצוין!'; alufim = 'אליפות!'; kadima = 'קדימה!'
  boss = 'זהירות, בוס!'; upgrade = 'שדרוג נשק!'; hit = 'אוי, נפגעת!'; win = 'ניצחת! כל הכבוד!'
  over = 'המשחק נגמר. ננסה שוב?'; boom = 'בום!'; chapter = 'פרק הושלם, יופי של משחק!'
}
$out = Join-Path (Get-Location) 'assets\voice'
New-Item -ItemType Directory -Force $out | Out-Null
foreach ($k in $lines.Keys) {
  $s = New-Object System.Speech.Synthesis.SpeechSynthesizer
  $s.SelectVoice('Microsoft Asaf')
  $s.Rate = 1
  $s.SetOutputToWaveFile((Join-Path $out "$k.wav"))
  $s.Speak($lines[$k])
  $s.Dispose()
  ffmpeg -y -v error -i (Join-Path $out "$k.wav") -ar 44100 -ac 2 -af "volume=2.5" -c:a libvorbis -q:a 5 (Join-Path $out "$k.ogg")
  Remove-Item (Join-Path $out "$k.wav")
}
