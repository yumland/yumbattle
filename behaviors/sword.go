package behaviors

import (
	"github.com/yumland/nbarena/bundle"
	"github.com/yumland/nbarena/draw"
	"github.com/yumland/nbarena/state"
	"github.com/yumland/nbarena/state/query"
	"github.com/yumland/pngsheet"
)

type SwordRange int

const (
	ShortSwordRange    SwordRange = 0
	WideSwordRange     SwordRange = 1
	LongSwordRange     SwordRange = 2
	VeryLongSwordRange SwordRange = 3
)

func slashAnimation(b *bundle.Bundle, r SwordRange) *pngsheet.Animation {
	switch r {
	case ShortSwordRange:
		return b.SlashSprites.ShortAnimation
	case WideSwordRange:
		return b.SlashSprites.WideAnimation
	case LongSwordRange:
		return b.SlashSprites.LongAnimation
	case VeryLongSwordRange:
		return b.SlashSprites.VeryLongAnimation
	}
	return nil
}

type Sword struct {
	Range  SwordRange
	Damage int
}

func (eb *Sword) Clone() state.EntityBehavior {
	return &Sword{
		eb.Range,
		eb.Damage,
	}
}

func swordTargetEntities(s *state.State, e *state.Entity, r SwordRange) []*state.Entity {
	x, y := e.TilePos.XY()
	dx := query.DXForward(e.IsFlipped)
	var entities []*state.Entity
	entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+dx, y))...)

	switch r {
	case WideSwordRange:
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+dx, y+1))...)
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+dx, y-1))...)
	case LongSwordRange:
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+2*dx, y))...)
	case VeryLongSwordRange:
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+2*dx, y))...)
		entities = append(entities, query.EntitiesAt(s, state.TilePosXY(x+3*dx, y))...)
	}

	return entities
}

func (eb *Sword) Step(e *state.Entity, s *state.State) {
	// Only hits while the slash is coming out.
	if e.BehaviorElapsedTime() == 9 {
		for _, entity := range swordTargetEntities(s, e, eb.Range) {
			if entity.FlashingTimeLeft == 0 {
				var h state.Hit
				h.FlashTime = state.DefaultFlashTime
				h.AddDamage(state.Damage{Base: eb.Damage})
				entity.CurrentHit.Merge(h)
				if !entity.Traits.CannotFlinch {
					entity.SetBehavior(&Flinch{})
				}
			}
		}
	} else if e.BehaviorElapsedTime() == 21 {
		e.SetBehavior(&Idle{})
	}
}

func (eb *Sword) Interrupts(e *state.Entity) state.EntityBehaviorInterrupts {
	return state.EntityBehaviorInterrupts{}
}

func (eb *Sword) Appearance(e *state.Entity, b *bundle.Bundle) draw.Node {
	rootNode := &draw.OptionsNode{}
	rootNode.Children = append(rootNode.Children, draw.ImageWithFrame(b.MegamanSprites.Image, b.MegamanSprites.SlashAnimation.Frames[e.BehaviorElapsedTime()]))

	swordNode := &draw.OptionsNode{Layer: 9}
	rootNode.Children = append(rootNode.Children, swordNode)
	swordNode.Children = append(swordNode.Children, draw.ImageWithFrame(b.SwordSprites.Image, b.SwordSprites.BaseAnimation.Frames[e.BehaviorElapsedTime()]))

	if e.BehaviorElapsedTime() >= 9 && e.BehaviorElapsedTime() < 19 {
		slashNode := &draw.OptionsNode{Layer: 8}
		slashNode.Opts.GeoM.Translate(float64(state.TileRenderedWidth), float64(-16))
		rootNode.Children = append(rootNode.Children, slashNode)

		slashAnim := slashAnimation(b, eb.Range)
		slashNode.Children = append(slashNode.Children, draw.ImageWithFrame(b.SlashSprites.SwordImage, slashAnim.Frames[e.BehaviorElapsedTime()-9]))
	}

	return rootNode
}
