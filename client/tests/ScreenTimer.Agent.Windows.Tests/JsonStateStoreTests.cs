using ScreenTimer.Agent.Core.Models;
using ScreenTimer.Agent.Windows.Storage;

namespace ScreenTimer.Agent.Windows.Tests;

public class JsonStateStoreTests : IDisposable
{
    private readonly string _tempDir;
    private readonly string _filePath;
    private readonly JsonStateStore _store;

    public JsonStateStoreTests()
    {
        _tempDir = Path.Combine(Path.GetTempPath(), "ScreenTimerTests_" + Guid.NewGuid().ToString("N"));
        Directory.CreateDirectory(_tempDir);
        _filePath = Path.Combine(_tempDir, "agent-state.json");
        _store = new JsonStateStore(_filePath);
    }

    public void Dispose()
    {
        if (Directory.Exists(_tempDir))
            Directory.Delete(_tempDir, recursive: true);
    }

    [Fact]
    public void Load_ReturnsNull_WhenFileDoesNotExist()
    {
        var result = _store.Load();
        Assert.Null(result);
    }

    [Fact]
    public void Save_CreatesDirectoryAndFile()
    {
        var subDir = Path.Combine(_tempDir, "sub", "dir");
        var subPath = Path.Combine(subDir, "state.json");
        var store = new JsonStateStore(subPath);

        store.Save(new AgentState { CurrentDate = "2026-04-01" });

        Assert.True(File.Exists(subPath));
    }

    [Fact]
    public void Save_And_Load_RoundTrips_FullState()
    {
        var state = new AgentState
        {
            LastForegroundExe = "game.exe",
            LastTickTime = new DateTimeOffset(2026, 4, 1, 10, 30, 0, TimeSpan.Zero),
            LastConfigPollTime = new DateTimeOffset(2026, 4, 1, 10, 29, 0, TimeSpan.Zero),
            LastUsageFlushTime = new DateTimeOffset(2026, 4, 1, 10, 29, 30, TimeSpan.Zero),
            CurrentDate = "2026-04-01",
            CurrentRules = new List<GroupRule>
            {
                new() { Name = "game.exe", Processes = new List<string> { "game.exe" }, DailyBudgetMinutes = 60 },
                new() { Name = "browser.exe", Processes = new List<string> { "browser.exe" }, DailyBudgetMinutes = 120 }
            }
        };
        state.Apps["game.exe"] = new AppUsageState
        {
            UsedTodaySeconds = 1500,
            PendingUploadSeconds = 45.5
        };
        state.Apps["browser.exe"] = new AppUsageState
        {
            UsedTodaySeconds = 300,
            PendingUploadSeconds = 10
        };

        _store.Save(state);
        var restored = _store.Load()!;

        Assert.NotNull(restored);
        Assert.Equal("game.exe", restored.LastForegroundExe);
        Assert.Equal(state.LastTickTime, restored.LastTickTime);
        Assert.Equal(state.LastConfigPollTime, restored.LastConfigPollTime);
        Assert.Equal(state.LastUsageFlushTime, restored.LastUsageFlushTime);
        Assert.Equal("2026-04-01", restored.CurrentDate);

        Assert.Equal(2, restored.CurrentRules.Count);
        Assert.Equal("game.exe", restored.CurrentRules[0].Name);
        Assert.Equal(60, restored.CurrentRules[0].DailyBudgetMinutes);

        Assert.Equal(2, restored.Apps.Count);
        var gameApp = restored.Apps["game.exe"];
        Assert.Equal(1500, gameApp.UsedTodaySeconds);
        Assert.Equal(45.5, gameApp.PendingUploadSeconds);
    }

    [Fact]
    public void Save_And_Load_RoundTrips_EmptyState()
    {
        var state = new AgentState();

        _store.Save(state);
        var restored = _store.Load()!;

        Assert.NotNull(restored);
        Assert.Null(restored.LastForegroundExe);
        Assert.Empty(restored.CurrentRules);
        Assert.Empty(restored.Apps);
    }

    [Fact]
    public void Save_Overwrites_ExistingFile()
    {
        _store.Save(new AgentState { CurrentDate = "2026-04-01" });
        _store.Save(new AgentState { CurrentDate = "2026-04-02" });

        var restored = _store.Load()!;
        Assert.Equal("2026-04-02", restored.CurrentDate);
    }

    [Fact]
    public void Save_DoesNotLeave_TmpFile()
    {
        _store.Save(new AgentState { CurrentDate = "2026-04-01" });

        Assert.False(File.Exists(_filePath + ".tmp"));
    }

    [Fact]
    public void Load_ReturnsNull_WhenFileContainsOldFormatState()
    {
        // Old format had AppRule with ExeName instead of GroupRule with Name/Processes
        var oldJson = """
            {
                "Apps": {
                    "game.exe": { "UsedTodaySeconds": 100, "PendingUploadSeconds": 10, "Sent10Min": true, "Sent5Min": false, "Sent1Min": false, "Exhausted": false }
                },
                "CurrentRules": [
                    { "ExeName": "game.exe", "DailyBudgetMinutes": 60 }
                ],
                "CurrentDate": "2026-04-01"
            }
            """;
        File.WriteAllText(_filePath, oldJson);

        var result = _store.Load();

        Assert.Null(result);
    }

    [Fact]
    public void Load_ReturnsNull_WhenFileContainsInvalidJson()
    {
        File.WriteAllText(_filePath, "not valid json {{{");

        var result = _store.Load();

        Assert.Null(result);
    }
}
