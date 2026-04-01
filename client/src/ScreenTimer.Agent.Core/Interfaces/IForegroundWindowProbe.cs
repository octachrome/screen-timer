using ScreenTimer.Agent.Core.Models;

namespace ScreenTimer.Agent.Core.Interfaces;

public interface IForegroundWindowProbe
{
    ForegroundSample Sample();
}
