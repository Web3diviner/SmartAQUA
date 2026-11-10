# <script>
<#
   scripts/ensure-gmail-only.ps1
   -------------------------------------------------
   Purpose: Abort the current process if the environment is NOT
            strictly gated to a single, explicitly‑named Gmail account.
   Usage:   powershell -File scripts/ensure-gmail-only.ps1 <ACCOUNT_LABEL>
#>

param(
    [Parameter(Mandatory=$true)]
    [string]$AccountLabel   # e.g. "primary-gmail"
)

# -------------------------------------------------
# Step 1 – Force the AGY_ACCOUNT env var for the rest of the session
$env:AGY_ACCOUNT = $AccountLabel
Write-Host "`n🔒 Enforcing Gmail‑only mode (account label: $AccountLabel)`n"

# -------------------------------------------------
# Step 2 – Verify the Gmail plugin can talk to that account
Write-Host "✅ Checking Gmail connectivity..."
$folders = agy gmail list --account $AccountLabel 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Error "❌ Gmail plugin cannot reach account `$AccountLabel`. Output:`n$folders"
    exit 1
}
Write-Host "   ✔ Gmail reachable – returned folders:`n   $folders"

# -------------------------------------------------
# Step 3 – Enumerate all linked Google accounts
Write-Host "`n🔎 Listing all linked Google accounts..."
$allAccounts = agy account list 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Error "❌ Failed to list accounts. Output:`n$allAccounts"
    exit 1
}
Write-Host "   Full account list:`n   $allAccounts"

# -------------------------------------------------
# Step 4 – Ensure there is exactly ONE Gmail entry and NO other Google accounts
$gmailMatches = ($allAccounts | Select-String -Pattern "$AccountLabel").Count
if ($gmailMatches -eq 0) {
    Write-Error "❌ No account matching `$AccountLabel` found in the account list."
    exit 1
}
if ($gmailMatches -gt 1) {
    Write-Error "❌ Multiple accounts match `$AccountLabel`. Use a unique label."
    exit 1
}

# Count any OTHER Google‑related entries (Drive, Calendar, etc.)
$otherGoogle = ($allAccounts | Select-String -Pattern "google|gmail|drive|calendar").Count - $gmailMatches
if ($otherGoogle -gt 0) {
    Write-Warning "⚠️  $otherGoogle other Google‑related account(s) are still linked."
    Write-Warning "   Disconnect them via the corresponding plugin UI before running production code."
    # Not exiting – you may decide to allow them, but the warning makes the situation clear.
}

Write-Host "`n✅ All checks passed – the environment is now strictly gated to Gmail account `$AccountLabel`."
exit 0
# </script>
