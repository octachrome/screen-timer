using ScreenTimer.Agent.Core.Dtos;

namespace ScreenTimer.Agent.Core.Engine;

public abstract record EngineCommand;

public sealed record ShowToastCommand(string ExeName, int RemainingMinutes) : EngineCommand;

public sealed record PushUsageCommand(UsagePushDto Payload) : EngineCommand;

public sealed record ForceCloseCommand(string ExeName) : EngineCommand;

public sealed record PersistStateCommand : EngineCommand;
