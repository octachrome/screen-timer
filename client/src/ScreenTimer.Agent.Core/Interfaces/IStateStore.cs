using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Interfaces;

public interface IStateStore
{
    AgentState? Load();
    void Save(AgentState state);
}
