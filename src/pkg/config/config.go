package config

import (
	_ "embed"
	"fmt"
	"reflect"
	"time"

	"gopkg.in/ini.v1"
)

//go:embed config.ini
var configFile []byte

// Config is the configuration of the game.
var Config config = func() (config config) {
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowBooleanKeys:          true,
		DebugFunc:                 Log,
		AllowNestedValues:         true,
		Insensitive:               true,
		SkipUnrecognizableLines:   true,
		UnescapeValueDoubleQuotes: true,
	}, configFile)
	ThrowError(err)

	// Set the block mode to false to speed up the loading.
	cfg.BlockMode = false
	ThrowError(cfg.MapTo(&config))

	// Sanitize the configuration.
	ThrowError(config.sanitize())
	return
}()

// config represents the configuration of the game.
type config struct {
	Bullet struct {
		CriticalHitChance       float64
		CriticalHitFactor       int
		DamageProgressAmplifier int
		Height                  float64
		InitialDamage           int
		ModifierProgressStep    int
		Speed                   float64
		SpeedDecayDuration      time.Duration
		Weight                  float64
		Width                   float64
	}

	Control struct {
		AnimationDuration                 time.Duration
		AudioEnabled                      *bool
		BackgroundAnimationEnabled        *bool
		CollisionDetectionVersion         EnvVariable[int]
		CriticalFramesPerSecondRate       float64
		Debug                             EnvVariable[bool]
		DesiredFramesPerSecondRate        float64
		DrawEnemyHitpointBars             EnvVariable[bool]
		DrawObjectLabels                  EnvVariable[bool]
		DrawSpaceshipDiscoveryProgressBar EnvVariable[bool]
		DrawSpaceshipExperienceBar        EnvVariable[bool]
		DrawSpaceshipShield               EnvVariable[bool]
		GodMode                           EnvVariable[bool]
		PlanetChoice                      EnvVariable[int]
		RepelEnemies                      EnvVariable[bool]
		SuspensionFrames                  int
	}

	Effect struct {
		BlastDuration time.Duration
	}

	Enemy struct {
		AccelerationProgress      float64
		BulletDamage              int
		ChargeSpeed               float64
		Count                     int
		CountProgressStep         int
		BerserkLikeliness         float64
		BerserkLikelinessProgress float64
		DefaultPenalty            int
		DefenseProgress           int
		FireCooldown              time.Duration
		FireLikeliness            float64
		FlashDecay                float64
		Height                    float64
		HitpointProgress          int
		InitialDefense            int
		InitialHitpoints          int
		InitialSpeed              float64
		LurkSpeed                 float64
		MaximumCount              int
		MaximumSpeed              float64
		PhaseRate                 float64
		Regenerate                *bool
		SpecialtyLikeliness       float64
		StrafeAmplitude           float64
		Width                     float64
		YetAgainAmplifier         float64

		Annihilator struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Annihilator"`

		Berserker struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Berserker"`

		Behemoth struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Behemoth"`

		Bulwark struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Bulwark"`

		Colossus struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Colossus"`

		Cloaked struct {
			Penalty       int
			SpeedModifier float64
		} `ini:"Enemy.Cloaked"`

		Dreadnought struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Dreadnought"`

		Freezer struct {
			Penalty int
		} `ini:"Enemy.Freezer"`

		Juggernaut struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Juggernaut"`

		Leviathan struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Leviathan"`

		Overlord struct {
			DefenseBoost    int
			HitpointsBoost  int
			Penalty         int
			SizeFactorBoost float64
			SpeedModifier   float64
		} `ini:"Enemy.Overlord"`
	}

	MessageBox struct {
		ChannelBufferSize    int
		ChannelLogThrottling time.Duration

		Messages struct {
			AllPlanetsDiscovered         TemplateString
			EnemyDestroyed               TemplateString
			EnemyHit                     TemplateString
			ExplainInterface             TemplateString
			GamePaused                   TemplateString
			GameStarted                  TemplateString
			GameOver                     TemplateString
			Greeting                     TemplateString
			HowToRestart                 TemplateString
			PerformanceDropped           TemplateString
			PerformanceImproved          TemplateString
			PlanetDiscovered             TemplateString
			PlanetImpactsSystem          TemplateString
			Prompt                       TemplateString
			ScoreBoardUpdated            TemplateString
			SpaceshipBoosted             TemplateString
			SpaceshipDowngradedByEnemy   TemplateString
			SpaceshipFrozen              TemplateString
			SpaceshipHijacked            TemplateString
			SpaceshipStillFrozen         TemplateString
			SpaceshipUpgradedByEnemyKill TemplateString
			SpaceshipUpgradedByTank      TemplateString
			WaitForScoreBoardUpdate      TemplateString
		} `ini:"MessageBox.Messages"`
	}

	Planet struct {
		DiscoveryCooldown    time.Duration
		DiscoveryProbability float64
		MaximumRadius        float64
		MinimumRadius        float64
		SpeedRatio           float64
		SpinRate             float64

		Impact struct {
			DefaultGravityStrength float64

			Mercury struct {
				BerserkLikelinessAmplifier float64
				Description                TemplateString
			} `ini:"Planet.Impact.Mercury"`

			Venus struct {
				Description             TemplateString
				SpaceshipDeceleration   float64
				TankLikelinessAmplifier float64
			} `ini:"Planet.Impact.Venus"`

			Earth struct {
				Description             TemplateString
				SpaceshipDeceleration   float64
				TankLikelinessAmplifier float64
			} `ini:"Planet.Impact.Earth"`

			Mars struct {
				BerserkLikelinessAmplifier float64
				Description                TemplateString
			} `ini:"Planet.Impact.Mars"`

			Jupiter struct {
				Description             TemplateString
				EnemyDefenseAmplifier   int
				EnemyHitpointsAmplifier int
			} `ini:"Planet.Impact.Jupiter"`

			Saturn struct {
				Description             TemplateString
				EnemyDefenseAmplifier   int
				EnemyHitpointsAmplifier int
			} `ini:"Planet.Impact.Saturn"`

			Uranus struct {
				Description                   TemplateString
				SpecialFoeLikelinessAmplifier float64
			} `ini:"Planet.Impact.Uranus"`

			Neptune struct {
				Description                   TemplateString
				SpecialFoeLikelinessAmplifier float64
			} `ini:"Planet.Impact.Neptune"`

			Pluto struct {
				BerserkLikelinessAmplifier    float64
				Description                   TemplateString
				SpecialFoeLikelinessAmplifier float64
			} `ini:"Planet.Impact.Pluto"`

			Sun struct {
				Description     TemplateString
				GravityStrength float64
			} `ini:"Planet.Impact.Sun"`

			BlackHole struct {
				Description              TemplateString
				GravityStrength          float64
				ObjectSizeDecayDuration  time.Duration
				SpaceshipDestroyedReason TemplateString
			} `ini:"Planet.Impact.BlackHole"`

			Supernova struct {
				Description     TemplateString
				GravityStrength float64
			} `ini:"Planet.Impact.Supernova"`
		} `ini:"Planet.Impact"`
	}

	Spaceship struct {
		Acceleration           float64
		AdmiralDamageAmplifier int
		BoostDuration          time.Duration
		BoostScaleSizeFactor   float64
		CannonProgress         int
		Cooldown               time.Duration
		DamageDuration         time.Duration
		Drag                   float64
		ExperienceScaler       float64
		FreezeDuration         time.Duration
		Height                 float64
		HijackDuration         time.Duration
		MaximumCannons         int
		MaximumLabelLength     int
		MaximumSpeed           float64
		MinimumSpeed           float64
		PointerOffsetFactor    float64
		ShieldChargeDuration   time.Duration
		Width                  float64
	}

	Star struct {
		Brightness         float64
		Count              int
		MinimumInnerRadius float64
		MinimumRadius      float64
		MinimumSpikes      float64
		MaximumInnerRadius float64
		MaximumRadius      float64
		MaximumSpikes      float64
		SpeedRatio         float64
	}
}

// Sanitize sanitizes the configuration.
// It calls the Sanitize method of each field that has one.
// The Sanitize method should have the following signature:
// func (Type) Sanitize() Type
func (cfg *config) sanitize() error {
	const methodName = "Sanitize"

	var sanitize func(reflect.Value) error
	sanitize = func(v reflect.Value) error {
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}

		if v.Kind() == reflect.Struct {
			for i := 0; i < v.NumField(); i++ {
				if err := sanitize(v.Field(i)); err != nil {
					return err
				}
			}
		}

		if !v.CanAddr() {
			return fmt.Errorf("cannot take the address of %v", v.Type())
		}

		method, ok := v.Type().MethodByName(methodName)
		if !ok {
			return nil
		}

		if method.Type.NumIn() != 1 || method.Type.NumOut() != 1 || !method.Type.Out(0).AssignableTo(v.Type()) {
			return fmt.Errorf("invalid signature for %[1]s method: %[2]v, expected: func (%[3]v) %[1]s() %[3]v",
				methodName, method.Type, v.Type())
		}

		v.Set(v.Addr().MethodByName(methodName).Call(nil)[0])
		return nil
	}

	return sanitize(reflect.ValueOf(cfg).Elem())
}
