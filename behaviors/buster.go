package behaviors

import (
	"github.com/murkland/nbarena/bundle"
	"github.com/murkland/nbarena/draw"
	"github.com/murkland/nbarena/input"
	"github.com/murkland/nbarena/state"
	"github.com/murkland/nbarena/state/query"
)

type Buster struct {
	BaseDamage   int
	IsPowerShot  bool
	isJammed     bool
	cooldownTime state.Ticks
}

func (eb *Buster) realElapsedTime(e *state.Entity) state.Ticks {
	t := e.BehaviorElapsedTime()
	if eb.IsPowerShot {
		t -= 5
	}
	return t
}

func (eb *Buster) Clone() state.EntityBehavior {
	return &Buster{
		eb.BaseDamage,
		eb.IsPowerShot,
		eb.isJammed,
		eb.cooldownTime,
	}
}

// Buster cooldown time:
var busterCooldownDurations = [][]state.Ticks{
	// d = 1, 2, 3, 4, 5, 6
	{5, 9, 13, 17, 21, 25}, // Lv1
	{4, 8, 11, 15, 18, 21}, // Lv2
	{4, 7, 10, 13, 16, 18}, // Lv3
	{3, 5, 7, 9, 11, 13},   // Lv4
	{3, 4, 5, 6, 7, 8},     // Lv5
}

func (eb *Buster) Step(e *state.Entity, s *state.State) {
	realElapsedTime := eb.realElapsedTime(e)

	if realElapsedTime == 5+eb.cooldownTime {
		e.SetBehavior(&Idle{})
		return
	}

	if realElapsedTime == 1 {
		_, d := query.FindNearestEntity(s, e.ID(), e.TilePos, e.IsAlliedWithAnswerer, e.IsFlipped, query.HorizontalDistance)
		eb.cooldownTime = busterCooldownDurations[0][d]

		x, y := e.TilePos.XY()
		if e.IsFlipped {
			x--
		} else {
			x++
		}

		shot := &state.Entity{
			TilePos:       state.TilePosXY(x, y),
			FutureTilePos: state.TilePosXY(x, y),

			IsFlipped:            e.IsFlipped,
			IsAlliedWithAnswerer: e.IsAlliedWithAnswerer,

			Traits: state.EntityTraits{
				CanStepOnHoleLikeTiles: true,
				IgnoresTileEffects:     true,
				CannotFlinch:           true,
				IgnoresTileOwnership:   true,
			},
		}

		damage := eb.BaseDamage
		if eb.IsPowerShot {
			damage *= 10
		}
		shot.SetBehavior(&busterShot{damage})
		s.AddEntity(shot)
	}

	if e.Intent.Direction != input.DirectionNone && realElapsedTime >= 5 {
		dir := e.Intent.Direction
		if e.ConfusedTimeLeft > 0 {
			dir = dir.FlipH().FlipV()
		}

		x, y := e.TilePos.XY()
		if dir&input.DirectionLeft != 0 {
			x--
		}
		if dir&input.DirectionRight != 0 {
			x++
		}
		if dir&input.DirectionUp != 0 {
			y--
		}
		if dir&input.DirectionDown != 0 {
			y++
		}

		if e.StartMove(state.TilePosXY(x, y), s.Field) {
			e.SetBehavior(&Teleport{})
		}
	}

}

func (eb *Buster) Appearance(e *state.Entity, b *bundle.Bundle) draw.Node {
	realElapsedTime := eb.realElapsedTime(e)

	if realElapsedTime < 0 {
		frame := b.MegamanSprites.IdleAnimation.Frames[0]
		return draw.ImageWithFrame(b.MegamanSprites.Image, frame)
	}

	rootNode := &draw.OptionsNode{}

	megamanBusterAnimTime := int(realElapsedTime)
	if megamanBusterAnimTime >= len(b.MegamanSprites.BusterAnimation.Frames) {
		megamanBusterAnimTime = len(b.MegamanSprites.BusterAnimation.Frames) - 1
	}
	rootNode.Children = append(rootNode.Children, draw.ImageWithFrame(b.MegamanSprites.Image, b.MegamanSprites.BusterAnimation.Frames[megamanBusterAnimTime]))

	busterFrames := b.BusterSprites.BaseAnimation
	busterAnimTime := int(realElapsedTime)
	if busterAnimTime >= len(busterFrames.Frames) {
		busterAnimTime = len(busterFrames.Frames) - 1
	}
	busterFrame := busterFrames.Frames[busterAnimTime]
	rootNode.Children = append(rootNode.Children, draw.ImageWithFrame(b.BusterSprites.Image, busterFrame))

	if !eb.isJammed {
		muzzleFlashAnimTime := int(realElapsedTime) - 1
		if muzzleFlashAnimTime > 0 && muzzleFlashAnimTime < len(b.MuzzleFlashSprites.Animations[0].Frames) {
			muzzleFlashNode := &draw.OptionsNode{Layer: 7}
			muzzleFlashNode.Opts.GeoM.Translate(float64(state.TileRenderedWidth), float64(-26))
			muzzleFlashFrame := b.MuzzleFlashSprites.Animations[0].Frames[muzzleFlashAnimTime]
			muzzleFlashNode.Children = append(muzzleFlashNode.Children, draw.ImageWithFrame(b.MuzzleFlashSprites.Image, muzzleFlashFrame))
			rootNode.Children = append(rootNode.Children, muzzleFlashNode)
		}
	}

	return rootNode
}

type busterShot struct {
	damage int
}

func (eb *busterShot) Flip() {
}

func (eb *busterShot) Clone() state.EntityBehavior {
	return &busterShot{
		eb.damage,
	}
}

func (eb *busterShot) Appearance(e *state.Entity, b *bundle.Bundle) draw.Node {
	return nil
}

func (eb *busterShot) Step(e *state.Entity, s *state.State) {
	if e.BehaviorElapsedTime()%2 == 1 {
		x, y := e.TilePos.XY()
		x += query.DXForward(e.IsFlipped)
		if !e.StartMove(state.TilePosXY(x, y), s.Field) {
			e.PerTickState.IsPendingDeletion = true
			return
		}
		e.FinishMove()
	}

	for _, target := range query.EntitiesAt(s, e.TilePos) {
		if target.IsAlliedWithAnswerer == e.IsAlliedWithAnswerer {
			continue
		}

		var h state.Hit
		h.AddDamage(state.Damage{Base: eb.damage})
		target.PerTickState.Hit.Merge(h)

		e.PerTickState.IsPendingDeletion = true
		return
	}
}
