using ScreenTimer.Agent.Core.Dtos;

namespace ScreenTimer.Agent.Core.Interfaces;

public interface IAgentApiClient
{
    Task<List<AppConfigDto>> GetConfigAsync(CancellationToken ct = default);
    Task PushUsageAsync(UsagePushDto push, CancellationToken ct = default);
}
