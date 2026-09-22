package tui

// camera rotates the flat 2D radial map in-plane (optional view aid).
type camera struct {
	yaw float64
}

const camYawStep = 0.40

func (c *camera) reset() {
	c.yaw = 0
}

func (c *camera) orbitYaw(dir float64) {
	c.yaw += dir * camYawStep
}
