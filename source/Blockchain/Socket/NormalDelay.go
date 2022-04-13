package Socket

import (
	"github.com/rs/zerolog/log"
	"gonum.org/v1/gonum/stat/distuv"
	"math/rand"
	"time"
)

type DelayType uint8

const (
	NoDelaySt DelayType = iota
	NormalDelaySt
	PoissonDelaySt
	FixeDelaySt
)

func StrToDelayType(behavior string) DelayType {
	switch behavior {
	case "NoDelay":
		return NoDelaySt
	case "Normal":
		return NormalDelaySt
	case "Poisson":
		return PoissonDelaySt
	case "Fix":
		return FixeDelaySt
	default:
		log.Error().Msgf("The string %s is not a delay type", behavior)
		return NoDelaySt
	}
}

type ProbaDelay interface {
	newDelay() float64
	newProbaDelay() ProbaDelay
	copy() ProbaDelay
	UpdateDelay(float642 float64) ProbaDelay
}

type NodeDelay struct {
	ProbaDelay
	// Set true to have same parameter on all nodes
	standard bool
}

type SocketDelay struct {
	ProbaDelay
}

func NewNodeDelay(delay ProbaDelay, isStandard bool) *NodeDelay {
	norm, ok_norm := delay.(NormalDelay)
	poisson, ok_poisson := delay.(PoissonDelay)
	if delay == nil || (ok_norm && norm.mean == 0) || (ok_poisson && poisson.parameter == 0) {
		return &NodeDelay{
			ProbaDelay: NoDelay{},
			standard:   true,
		}
	}
	return &NodeDelay{delay, isStandard}
}

func (node NodeDelay) NewSocketDelay() *SocketDelay {
	if node.standard {
		return &SocketDelay{node.copy()}
	} else {
		return &SocketDelay{node.newProbaDelay()}
	}
}

//func (socket *SocketDelay) updateDelay(parameter float64) {
//	socket.ProbaDelay = socket.ProbaDelay.updateDelay(parameter)
//}

type NormalDelay struct {
	stdDev float64
	mean   float64
}

func NewNormalDelay(mean float64, stdDev float64) NormalDelay {
	return NormalDelay{stdDev: stdDev, mean: mean}
}

func (normal NormalDelay) newDelay() float64 {
	var delay float64
	for delay <= 0 {
		delay = rand.NormFloat64()*normal.stdDev + normal.mean
	}
	return delay
}

func (normal NormalDelay) newProbaDelay() ProbaDelay {
	return NormalDelay{
		stdDev: normal.stdDev,
		mean:   normal.newDelay(),
	}
}

func (normal NormalDelay) copy() ProbaDelay {
	return NormalDelay{
		stdDev: normal.stdDev,
		mean:   normal.mean,
	}
}

func (normal NormalDelay) UpdateDelay(newMean float64) ProbaDelay {
	normal.mean = newMean
	return normal
}

func (socket SocketDelay) GetSleepNewDelay() time.Duration {
	return time.Duration(socket.newDelay()) * time.Millisecond
}

func (socket SocketDelay) SleepNewDelay() {
	delay := socket.newDelay()
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

type NoDelay struct{}

func (delay NoDelay) newDelay() float64 {
	return 0
}

func (delay NoDelay) newProbaDelay() ProbaDelay {
	return NoDelay{}
}

func (delay NoDelay) copy() ProbaDelay {
	return NoDelay{}
}

func (delay NoDelay) UpdateDelay(float64) ProbaDelay { return delay }

type PoissonDelay struct {
	parameter  float64
	poissonLaw distuv.Poisson
}

func NewPoissonDelay(parameter float64) PoissonDelay {
	return PoissonDelay{parameter, distuv.Poisson{
		Lambda: parameter,
	}}
}

func (poisson PoissonDelay) newDelay() float64 {
	return poisson.poissonLaw.Rand()
}

func (poisson PoissonDelay) newProbaDelay() ProbaDelay {
	return NewPoissonDelay(poisson.newDelay())
}

func (poisson PoissonDelay) copy() ProbaDelay {
	return NewPoissonDelay(poisson.parameter)
}

func (poisson PoissonDelay) UpdateDelay(parameter float64) ProbaDelay {
	poisson.parameter = parameter
	poisson.poissonLaw = distuv.Poisson{
		Lambda: parameter,
	}
	return poisson
}

type fixeDelay struct {
	parameter float64
}

func NewFixeDelay(parameter float64) *fixeDelay {
	return &fixeDelay{parameter: parameter}
}

func (delay fixeDelay) newDelay() float64 {
	return delay.parameter
}

func (delay fixeDelay) newProbaDelay() ProbaDelay {
	return fixeDelay{parameter: delay.parameter}
}

func (delay fixeDelay) copy() ProbaDelay {
	return fixeDelay{parameter: delay.parameter}
}

func (delay fixeDelay) UpdateDelay(parameter float64) ProbaDelay {
	delay.parameter = parameter
	return delay
}
