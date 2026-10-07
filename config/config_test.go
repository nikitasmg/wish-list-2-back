package config

import "testing"

func TestLoadConfig_SantaBotTokenFallsBackToBotToken(t *testing.T) {
	cases := []struct{ name, santa, bot, want string }{
		{"empty santa", "", "x", "x"},
		{"santa set", "y", "x", "y"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SANTA_BOT_TOKEN", tc.santa)
			t.Setenv("BOT_TOKEN", tc.bot)
			cfg, err := LoadConfig()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Notify.TelegramBotToken != tc.want {
				t.Fatalf("got %q, want %q", cfg.Notify.TelegramBotToken, tc.want)
			}
		})
	}
}
