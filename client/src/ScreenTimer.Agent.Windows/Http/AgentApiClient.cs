using System.Net.Http.Json;
using ScreenTimer.Agent.Core.Dtos;
using ScreenTimer.Agent.Core.Interfaces;

namespace ScreenTimer.Agent.Windows.Http;

public sealed class AgentApiClient(HttpClient http) : IAgentApiClient
{
    public async Task<AgentConfigResponseDto> GetConfigAsync(CancellationToken ct = default)
    {
        var response = await http.GetAsync("/api/agent/config", ct);
        response.EnsureSuccessStatusCode();
        return await response.Content.ReadFromJsonAsync<AgentConfigResponseDto>(ct) ?? new();
    }

    public async Task PushUsageAsync(UsagePushDto push, CancellationToken ct = default)
    {
        var response = await http.PostAsJsonAsync("/api/agent/usage", push, ct);
        response.EnsureSuccessStatusCode();
    }
}
