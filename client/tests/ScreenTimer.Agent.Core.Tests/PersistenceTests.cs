using System.Text.Json;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class PersistenceTests
{
    [Fact]
    public void State_RoundTrips_Through_Json()
    {
        var state = new AgentState
        {
            LastForegroundExe = "game.exe",
            LastTickTime = new DateTimeOffset(2026, 4, 1, 10, 30, 0, TimeSpan.Zero),
            LastConfigPollTime = new DateTimeOffset(2026, 4, 1, 10, 29, 0, TimeSpan.Zero),
            LastUsageFlushTime = new DateTimeOffset(2026, 4, 1, 10, 29, 30, TimeSpan.Zero),
            CurrentDate = "2026-04-01",
            CurrentRules = new List<AppRule>
            {
                new() { ExeName = "game.exe", DailyBudgetMinutes = 60 },
                new() { ExeName = "browser.exe", DailyBudgetMinutes = 120 }
            }
        };
        state.Apps["game.exe"] = new AppUsageState
        {
            UsedTodaySeconds = 1500,
            PendingUploadSeconds = 45.5,
            Sent10Min = true,
            Sent5Min = false,
            Sent1Min = false,
            Exhausted = false
        };
        state.Apps["browser.exe"] = new AppUsageState
        {
            UsedTodaySeconds = 300,
            PendingUploadSeconds = 10,
            Sent10Min = false,
            Sent5Min = false,
            Sent1Min = false,
            Exhausted = false
        };

        var json = JsonSerializer.Serialize(state);
        var restored = JsonSerializer.Deserialize<AgentState>(json)!;

        Assert.Equal(state.LastForegroundExe, restored.LastForegroundExe);
        Assert.Equal(state.LastTickTime, restored.LastTickTime);
        Assert.Equal(state.LastConfigPollTime, restored.LastConfigPollTime);
        Assert.Equal(state.LastUsageFlushTime, restored.LastUsageFlushTime);
        Assert.Equal(state.CurrentDate, restored.CurrentDate);

        Assert.Equal(state.CurrentRules.Count, restored.CurrentRules.Count);
        for (int i = 0; i < state.CurrentRules.Count; i++)
        {
            Assert.Equal(state.CurrentRules[i].ExeName, restored.CurrentRules[i].ExeName);
            Assert.Equal(state.CurrentRules[i].DailyBudgetMinutes, restored.CurrentRules[i].DailyBudgetMinutes);
        }

        Assert.Equal(state.Apps.Count, restored.Apps.Count);
        foreach (var (key, original) in state.Apps)
        {
            Assert.True(restored.Apps.ContainsKey(key), $"Missing key: {key}");
            var restoredApp = restored.Apps[key];
            Assert.Equal(original.UsedTodaySeconds, restoredApp.UsedTodaySeconds);
            Assert.Equal(original.PendingUploadSeconds, restoredApp.PendingUploadSeconds);
            Assert.Equal(original.Sent10Min, restoredApp.Sent10Min);
            Assert.Equal(original.Sent5Min, restoredApp.Sent5Min);
            Assert.Equal(original.Sent1Min, restoredApp.Sent1Min);
            Assert.Equal(original.Exhausted, restoredApp.Exhausted);
        }
    }

    [Fact]
    public void Empty_State_RoundTrips_Through_Json()
    {
        var state = new AgentState();

        var json = JsonSerializer.Serialize(state);
        var restored = JsonSerializer.Deserialize<AgentState>(json)!;

        Assert.Null(restored.LastForegroundExe);
        Assert.Equal(default, restored.LastTickTime);
        Assert.Equal(default, restored.LastConfigPollTime);
        Assert.Equal(default, restored.LastUsageFlushTime);
        Assert.Equal("", restored.CurrentDate);
        Assert.Empty(restored.CurrentRules);
        Assert.Empty(restored.Apps);
    }

    [Fact]
    public void State_With_Exhausted_App_RoundTrips_Through_Json()
    {
        var state = new AgentState
        {
            CurrentDate = "2026-04-01",
            CurrentRules = new List<AppRule>
            {
                new() { ExeName = "game.exe", DailyBudgetMinutes = 30 }
            }
        };
        state.Apps["game.exe"] = new AppUsageState
        {
            UsedTodaySeconds = 1850,
            PendingUploadSeconds = 50,
            Sent10Min = true,
            Sent5Min = true,
            Sent1Min = true,
            Exhausted = true
        };

        var json = JsonSerializer.Serialize(state);
        var restored = JsonSerializer.Deserialize<AgentState>(json)!;

        var restoredApp = restored.Apps["game.exe"];
        Assert.True(restoredApp.Sent10Min);
        Assert.True(restoredApp.Sent5Min);
        Assert.True(restoredApp.Sent1Min);
        Assert.True(restoredApp.Exhausted);
        Assert.Equal(1850, restoredApp.UsedTodaySeconds);
        Assert.Equal(50, restoredApp.PendingUploadSeconds);
    }
}
