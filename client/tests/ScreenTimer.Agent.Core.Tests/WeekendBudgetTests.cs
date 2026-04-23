using ScreenTimer.Agent.Core.Engine;
using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Tests;

public class WeekendBudgetTests
{
    // Monday Jan 6 2025 is a known weekday
    private static readonly DateTimeOffset Monday = new(2025, 1, 6, 12, 0, 0, TimeSpan.Zero);
    // Saturday Jan 4 2025 is a known weekend
    private static readonly DateTimeOffset Saturday = new(2025, 1, 4, 12, 0, 0, TimeSpan.Zero);

    private static List<GroupRule> MakeRules(int weekdayBudget, int weekendBudget) => new()
    {
        new GroupRule
        {
            Name = "Games",
            Processes = new List<string> { "game.exe" },
            DailyBudgetMinutes = weekdayBudget,
            WeekendBudgetMinutes = weekendBudget
        }
    };

    [Fact]
    public void Tick_WeekdayUsesWeekdayBudget()
    {
        var state = new AgentState();
        var rules = MakeRules(60, 30);

        // First tick: apply rules, start tracking on a Monday
        var t0 = Monday;
        AgentEngine.Tick(state, new ForegroundSample("game.exe", t0), rules);

        // Second tick: 59 minutes later (within 60-min weekday budget)
        var t1 = t0.AddMinutes(59);
        var result = AgentEngine.Tick(state, new ForegroundSample("game.exe", t1), null);

        // Should NOT have a ForceClose — still within weekday budget
        Assert.DoesNotContain(result.Commands, c => c is ForceCloseCommand);
    }

    [Fact]
    public void Tick_WeekdayEnforcesWeekdayBudget()
    {
        var state = new AgentState();
        var rules = MakeRules(60, 30);

        var t0 = Monday;
        AgentEngine.Tick(state, new ForegroundSample("game.exe", t0), rules);

        // 61 minutes later — exceeds 60-min weekday budget
        var t1 = t0.AddMinutes(61);
        var result = AgentEngine.Tick(state, new ForegroundSample("game.exe", t1), null);

        Assert.Contains(result.Commands, c => c is ForceCloseCommand);
    }

    [Fact]
    public void Tick_WeekendUsesWeekendBudget()
    {
        var state = new AgentState();
        var rules = MakeRules(60, 30);

        // First tick on Saturday
        var t0 = Saturday;
        AgentEngine.Tick(state, new ForegroundSample("game.exe", t0), rules);

        // 31 minutes later — exceeds 30-min weekend budget
        var t1 = t0.AddMinutes(31);
        var result = AgentEngine.Tick(state, new ForegroundSample("game.exe", t1), null);

        Assert.Contains(result.Commands, c => c is ForceCloseCommand);
    }

    [Fact]
    public void Tick_WeekendBudgetNotEnforcedBeforeExpiry()
    {
        var state = new AgentState();
        var rules = MakeRules(60, 30);

        var t0 = Saturday;
        AgentEngine.Tick(state, new ForegroundSample("game.exe", t0), rules);

        // 29 minutes — within 30-min weekend budget
        var t1 = t0.AddMinutes(29);
        var result = AgentEngine.Tick(state, new ForegroundSample("game.exe", t1), null);

        Assert.DoesNotContain(result.Commands, c => c is ForceCloseCommand);
    }

    [Fact]
    public void Tick_WeekendNotificationsFireAtCorrectThresholds()
    {
        var state = new AgentState();
        var rules = MakeRules(60, 30); // 30-min weekend budget

        var t0 = Saturday;
        AgentEngine.Tick(state, new ForegroundSample("game.exe", t0), rules);

        // 20 minutes in → 10 min remaining → should fire 10-min notification
        var t1 = t0.AddMinutes(20);
        var result1 = AgentEngine.Tick(state, new ForegroundSample("game.exe", t1), null);
        Assert.Contains(result1.Commands, c => c is ShowToastCommand st && st.RemainingMinutes == 10);

        // 25 minutes in → 5 min remaining → should fire 5-min notification
        var t2 = t0.AddMinutes(25);
        var result2 = AgentEngine.Tick(state, new ForegroundSample("game.exe", t2), null);
        Assert.Contains(result2.Commands, c => c is ShowToastCommand st && st.RemainingMinutes == 5);

        // 29 minutes in → 1 min remaining → should fire 1-min notification
        var t3 = t0.AddMinutes(29);
        var result3 = AgentEngine.Tick(state, new ForegroundSample("game.exe", t3), null);
        Assert.Contains(result3.Commands, c => c is ShowToastCommand st && st.RemainingMinutes == 1);
    }
}
