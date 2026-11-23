$source = Split-Path -Parent $PSScriptRoot

Describe 'Safe push index boundary' {
  BeforeEach {
    $fixture = Join-Path $TestDrive ([guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path (Join-Path $fixture 'scripts') -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $source 'safe-push.ps1') -Destination $fixture
    Copy-Item -LiteralPath (Join-Path $source 'scripts/safe_push.py') -Destination (Join-Path $fixture 'scripts')
    & git -C $fixture init -q
    & git -C $fixture config user.name 'QC Fixture'
    & git -C $fixture config user.email 'fixture@example.invalid'
    Set-Content -LiteralPath (Join-Path $fixture '.gitignore') -Value @('*.local','!safe.local')
    Set-Content -LiteralPath (Join-Path $fixture 'safe.local') -Value 'versioned source'
    & git -C $fixture add -- .gitignore safe.local
    & git -C $fixture commit -qm 'fixture baseline'
    $head = & git -C $fixture rev-parse HEAD
    $scriptPath = Join-Path $fixture 'safe-push.ps1'
  }

  It 'honors Git negation and checks without staging untracked files' {
    Set-Content -LiteralPath (Join-Path $fixture 'untracked.txt') -Value 'fixture'
    $result = & pwsh -NoProfile -File $scriptPath 2>&1
    $LASTEXITCODE | Should Be 0
    ($result -join ' ') | Should Match 'SAFE_PUSH_CHECK_PASS'
    (& git -C $fixture status --porcelain -- untracked.txt) | Should Be '?? untracked.txt'
    (& git -C $fixture rev-parse HEAD) | Should Be $head
  }

  It 'reports an ignored tracked file without removing it' {
    Set-Content -LiteralPath (Join-Path $fixture 'blocked.local') -Value 'fixture'
    & git -C $fixture add -f -- blocked.local
    $before = & git -C $fixture ls-files --stage
    $result = & pwsh -NoProfile -File $scriptPath 2>&1
    $LASTEXITCODE | Should Be 1
    ($result -join ' ') | Should Match 'tracked path'
    ((& git -C $fixture ls-files --stage) -join ';') | Should Be ($before -join ';')
    (& git -C $fixture rev-parse HEAD) | Should Be $head
  }

  It 'scans the staged blob even if the worktree copy was cleaned' {
    $file = Join-Path $fixture 'candidate.txt'
    $synthetic = 'sk-' + ('X' * 40)
    Set-Content -LiteralPath $file -Value $synthetic
    & git -C $fixture add -- candidate.txt
    Set-Content -LiteralPath $file -Value 'clean worktree'
    $result = & pwsh -NoProfile -File $scriptPath 2>&1
    $LASTEXITCODE | Should Be 1
    ($result -join ' ') | Should Match 'possible credential'
    ($result -join ' ') | Should Not Match $synthetic
    (& git -C $fixture rev-parse HEAD) | Should Be $head
  }

  It 'commits explicitly staged content in the fixture only' {
    Set-Content -LiteralPath (Join-Path $fixture 'safe.local') -Value 'reviewed update'
    Set-Content -LiteralPath (Join-Path $fixture 'untracked.txt') -Value 'fixture'
    & git -C $fixture add -- safe.local
    $result = & pwsh -NoProfile -File $scriptPath -Commit -NoPush -Message 'fixture update' 2>&1
    $LASTEXITCODE | Should Be 0
    ($result -join ' ') | Should Match 'SAFE_PUSH_COMMITTED'
    (& git -C $fixture rev-parse 'HEAD^') | Should Be $head
    (& git -C $fixture show 'HEAD:safe.local') | Should Be 'reviewed update'
    (& git -C $fixture status --porcelain -- untracked.txt) | Should Be '?? untracked.txt'
  }
}
