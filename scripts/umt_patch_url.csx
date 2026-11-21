// Hero Siege backend URL patcher for UndertaleModTool.
//
// Replaces the legacy hardcoded backend URLs in data.win with a new base URL.
// Only STRG string literals are touched: the game code is not decompiled or
// recompiled, so nothing else in the data.win changes.
//
// ---------------------------------------------------------------------------
// GUI usage
// ---------------------------------------------------------------------------
//   1. Open data.win in UndertaleModTool.
//   2. Scripts -> Run script... -> select this file.
//   3. Enter the new backend base URL (for example http://localhost:8080/).
//
// ---------------------------------------------------------------------------
// CLI usage
// ---------------------------------------------------------------------------
//   UndertaleModCli load data.win -s scripts/umt_patch_url.csx -o data_patched.win
//
// You can also provide the URL through the environment instead of typing it:
//
//   HERO_SIEGE_BACKEND_URL=http://10.0.0.5:8080/ \
//     UndertaleModCli load data.win -s scripts/umt_patch_url.csx -o data_patched.win
// ---------------------------------------------------------------------------

using System;
using System.IO;
using System.Collections.Generic;
using UndertaleModLib;
using UndertaleModLib.Models;

EnsureDataLoaded();

const string EnvVar = "HERO_SIEGE_BACKEND_URL";
const string DefaultBaseUrl = "http://localhost:8080/";

if (Data.Strings is null)
{
    ScriptError("This data file has no STRG (string) chunk.", "Nothing to patch");
    return;
}

string newBase = Environment.GetEnvironmentVariable(EnvVar);
if (string.IsNullOrWhiteSpace(newBase))
{
    newBase = SimpleTextInput(
        "Hero Siege backend URL",
        "Enter the replacement backend base URL (e.g. http://localhost:8080/):",
        DefaultBaseUrl,
        false);
}

if (string.IsNullOrWhiteSpace(newBase))
    newBase = DefaultBaseUrl;

newBase = newBase.Trim();

if (!newBase.StartsWith("http://", StringComparison.OrdinalIgnoreCase) &&
    !newBase.StartsWith("https://", StringComparison.OrdinalIgnoreCase))
{
    newBase = "http://" + newBase;
}

if (!newBase.EndsWith("/"))
    newBase += "/";

// Old prefix -> replacement prefix. These are the known hardcoded backend URLs
// from Hero Siege's data.win. Paths are preserved after the prefix, so for
// example "...:8080/ping.php" becomes "<newBase>ping.php".
//
// The localhost entry is included so the script can be re-run after a previous
// localhost patch (for example to move from localhost to a LAN IP).
var replacements = new (string OldPrefix, string NewPrefix)[]
{
    // Legacy LAN/dev backend base URLs, with and without the :8080 port.
    ("http://10.0.0.212:8080/", newBase),
    ("http://10.0.0.212/", newBase),
    ("http://10.0.0.232:8080/", newBase),
    ("http://10.0.0.232/", newBase),

    // Previously patched localhost base URL.
    ("http://localhost:8080/", newBase),

    // Legacy production backend base URLs.
    ("https://www.autoseries.net/ingame/herosiegebeta/", newBase),
    ("https://www.autoseries.net/ingame/herosiege/", newBase),

    // Standalone ping endpoint (production host).
    ("http://www.autoseries.net/ingame/ping.php", newBase + "ping.php"),

    // Redeem code endpoint (the "?code=" suffix is preserved).
    ("https://herosiege.autoseries.net/ingame/herosiege/code_redeem.php", newBase + "code_redeem.php"),
};

var changes = new List<string>();
int patched = 0;

foreach (var str in Data.Strings)
{
    if (str?.Content is null)
        continue;

    string content = str.Content;
    foreach (var (oldPrefix, newPrefix) in replacements)
    {
        if (content.StartsWith(oldPrefix, StringComparison.Ordinal))
        {
            string newContent = newPrefix + content.Substring(oldPrefix.Length);
            if (newContent == content)
                break;

            changes.Add(content + "  ->  " + newContent);
            str.Content = newContent;
            patched++;
            break;
        }
    }
}

if (patched == 0)
{
    ScriptError("No known Hero Siege backend URLs were found in this data file.", "Nothing to patch");
    return;
}

ScriptMessage("Patched " + patched + " string literal(s):\n\n" + string.Join("\n", changes));

if (CanSave)
{
    if (ScriptQuestion("Save the patched data.win now? A .bak backup will be created."))
    {
        SavePatchedData();
        ScriptMessage("Saved patched data.win to:\n" + FilePath);
    }
    else
    {
        ScriptMessage("Changes were applied in memory only. Use File -> Save to persist them.");
    }
}
else
{
    ScriptMessage("The patched data is in memory. (CLI writes it automatically when -o <output> is provided; in the GUI use File -> Save.)");
}

void SavePatchedData()
{
    if (string.IsNullOrEmpty(FilePath))
    {
        ScriptError("No file path is set; save the file manually first.", "Cannot save");
        return;
    }

    try
    {
        string backupPath = FilePath + ".bak";
        if (!File.Exists(backupPath))
            File.Copy(FilePath, backupPath, false);

        using FileStream stream = new FileStream(FilePath, FileMode.Create, FileAccess.Write);
        UndertaleIO.Write(stream, Data);
    }
    catch (Exception ex)
    {
        ScriptError("Failed to save patched data.win:\n\n" + ex, "Save error");
    }
}
