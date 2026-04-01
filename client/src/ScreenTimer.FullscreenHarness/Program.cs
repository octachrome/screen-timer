namespace ScreenTimer.FullscreenHarness;

static class Program
{
    [STAThread]
    static void Main(string[] args)
    {
        string mode = "windowed";
        int resistCloseSeconds = 0;

        for (int i = 0; i < args.Length; i++)
        {
            switch (args[i])
            {
                case "--mode" when i + 1 < args.Length:
                    mode = args[++i].ToLowerInvariant();
                    break;
                case "--resist-close" when i + 1 < args.Length:
                    resistCloseSeconds = int.Parse(args[++i]);
                    break;
            }
        }

        if (mode is not ("windowed" or "borderless" or "exclusive"))
        {
            Console.Error.WriteLine($"Unknown mode: {mode}. Use windowed, borderless, or exclusive.");
            return;
        }

        Console.WriteLine($"Launching harness: mode={mode}, resist-close={resistCloseSeconds}s");

        ApplicationConfiguration.Initialize();
        Application.Run(new HarnessForm(mode, resistCloseSeconds));
    }
}
