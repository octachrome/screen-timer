using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Engine;

public sealed class EngineResult
{
    public required AgentState UpdatedState { get; init; }
    public List<EngineCommand> Commands { get; init; } = new();
}
