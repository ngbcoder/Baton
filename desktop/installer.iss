[Setup]
AppName=Baton
AppVersion=0.1.0
DefaultDirName={autopf}\Baton
DefaultGroupName=Baton
OutputBaseFilename=baton-setup
Compression=lzma
SolidCompression=yes
ArchitecturesInstallIn64BitMode=x64

[Files]
Source: "bin\baton.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Baton"; Filename: "{app}\baton.exe"
Name: "{autodesktop}\Baton"; Filename: "{app}\baton.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional icons:"

[Run]
Filename: "{app}\baton.exe"; Description: "Launch Baton"; Flags: nowait postinstall skipifsilent