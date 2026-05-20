import Phaser from 'phaser'

type Cell = number

const COLS = 10
const ROWS = 20
const CELL = 28

export const GAME_BASE_WIDTH = COLS * CELL

const SHAPES: number[][][] = [
  // I
  [[0,1,0,0],[0,1,0,0],[0,1,0,0],[0,1,0,0]],
  // J
  [[1,0,0],[1,1,1],[0,0,0]],
  // L
  [[0,0,1],[1,1,1],[0,0,0]],
  // O
  [[1,1],[1,1]],
  // S
  [[0,1,1],[1,1,0],[0,0,0]],
  // T
  [[0,1,0],[1,1,1],[0,0,0]],
  // Z
  [[1,1,0],[0,1,1],[0,0,0]]
]

const COLORS = [0x00ffff, 0x0000ff, 0xffa500, 0xffff00, 0x00ff00, 0x800080, 0xff0000]

function rotate(shape: number[][]) {
  const n = shape.length
  const res = Array.from({length: n}, () => Array(n).fill(0))
  for (let r=0;r<n;r++) for (let c=0;c<n;c++) res[c][n-1-r] = shape[r][c]
  return res
}

export class TetrisScene extends Phaser.Scene {
  board: number[][]
  shape: number[][] | null = null
  shapeX = 3
  shapeY = 0
  shapeColor = 0
  dropTimer = 0
  dropInterval = 600
  graphics: Phaser.GameObjects.Graphics | null = null
  score = 0
  totalLinesCleared = 0
  running = false
  // next piece preview
  nextLabel: Phaser.GameObjects.Text | null = null
  nextShapeIndex = 0
  nextShape: number[][] = []

  constructor() {
    super({key: 'TetrisScene'})
    this.board = Array.from({length: ROWS}, () => Array(COLS).fill(0))
    this.nextShapeIndex = Phaser.Math.Between(0, SHAPES.length - 1)
    this.nextShape = SHAPES[this.nextShapeIndex].map(r => r.slice())
  }

  preload() {
    // load reward images from the dev server root (frontend/imgs)
    this.load.image('knife', '/imgs/knife.png')
    this.load.image('pistol', '/imgs/pistol.png')
    this.load.image('bomb', '/imgs/bomb.png')
  }

  create() {
    this.graphics = this.add.graphics()
    this.nextLabel = this.add.text(0, 6, 'NEXT', { color: '#888', fontSize: '10px' }).setOrigin(1, 0)
    this.spawnShape()
    this.input.keyboard.on('keydown', this.handleKey, this)
    this.events.on('useItem', this.useItem, this)
    console.log('TetrisScene: create complete, graphics and input ready')
  }

  update(time: number) {
    if (!this.graphics) return
    if (!this.running) return
    if (time > this.dropTimer + this.dropInterval) {
      this.dropTimer = time
      this.moveDown()
    }
    this.render()
  }

  startGame() {
    this.running = true
    this.dropTimer = this.time.now
    // ensure input is active
  }

  pauseGame() {
    this.running = false
  }

  spawnShape() {
    const idx = this.nextShapeIndex
    this.shape = this.nextShape
    const n = this.shape.length
    this.shapeX = Math.floor((COLS - n)/2)
    this.shapeY = 0
    this.shapeColor = COLORS[idx]
    // generate next piece
    this.nextShapeIndex = Phaser.Math.Between(0, SHAPES.length - 1)
    this.nextShape = SHAPES[this.nextShapeIndex].map(r => r.slice())
    if (this.collides(this.shapeX, this.shapeY, this.shape)) {
      this.scene.pause()
      this.add.text(40, 200, 'Game Over', {color:'#ff0000', fontSize:'24px'})
    }
  }

  handleKey(e: KeyboardEvent) {
    if (!this.shape) return
    if (e.code === 'ArrowLeft') this.tryMove(this.shapeX -1, this.shapeY, this.shape)
    else if (e.code === 'ArrowRight') this.tryMove(this.shapeX +1, this.shapeY, this.shape)
    else if (e.code === 'ArrowDown') this.moveDown()
    else if (e.code === 'Space') this.hardDrop()
    else if (e.code === 'ArrowUp') this.rotateShape()
  }

  tryMove(x:number,y:number,shape:number[][]) {
    if (!this.collides(x,y,shape)) { this.shapeX = x; this.shapeY = y }
  }

  moveDown() {
    if (!this.shape) return
    if (!this.collides(this.shapeX, this.shapeY+1, this.shape)) {
      this.shapeY += 1
    } else {
      this.lock()
      this.clearLines()
      this.spawnShape()
    }
  }

  hardDrop() {
    while (this.shape && !this.collides(this.shapeX, this.shapeY+1, this.shape)) this.shapeY++
    this.lock()
    this.clearLines()
    this.spawnShape()
  }

  rotateShape() {
    if (!this.shape) return
    const rotated = rotate(this.shape)
    if (!this.collides(this.shapeX, this.shapeY, rotated)) this.shape = rotated
  }

  useItem(item: string) {
    switch (item) {
      case 'bomb': this.bombClear(); break
      case 'pistol': this.pistolClear(); break
      case 'knife': this.knifeClear(); break
    }
    this.render()
  }

  // bomb: unconditionally clear bottom 3 rows, shift board down
  private bombClear() {
    const count = Math.min(3, ROWS)
    for (let i = 0; i < count; i++) {
      this.board.pop()
      this.board.unshift(Array(COLS).fill(0))
    }
    this.score += count * 10
    this.events.emit('scoreChanged', this.score)
  }

  // pistol: unconditionally clear bottom 1 row, shift board down
  private pistolClear() {
    this.board.pop()
    this.board.unshift(Array(COLS).fill(0))
    this.score += 10
    this.events.emit('scoreChanged', this.score)
  }

  // knife: remove 2 random blocks from the topmost non-empty row
  private knifeClear() {
    for (let r = 0; r < ROWS; r++) {
      const filled: number[] = []
      for (let c = 0; c < COLS; c++) {
        if (this.board[r][c] !== 0) filled.push(c)
      }
      if (filled.length === 0) continue
      const count = Math.min(5, filled.length)
      for (let i = 0; i < count; i++) {
        const idx = Phaser.Math.Between(0, filled.length - 1)
        this.board[r][filled[idx]] = 0
        filled.splice(idx, 1)
      }
      break
    }
  }

  collides(x:number,y:number,shape:number[][]) {
    const n = shape.length
    for (let r=0;r<n;r++) for (let c=0;c<n;c++) {
      if (shape[r][c]) {
        const bx = x + c
        const by = y + r
        if (bx < 0 || bx >= COLS || by >= ROWS) return true
        if (by >= 0 && this.board[by][bx]) return true
      }
    }
    return false
  }

  lock() {
    if (!this.shape) return
    const n = this.shape.length
    for (let r=0;r<n;r++) for (let c=0;c<n;c++) if (this.shape[r][c]) {
      const bx = this.shapeX + c
      const by = this.shapeY + r
      if (by >= 0 && by < ROWS && bx >=0 && bx < COLS) this.board[by][bx] = this.shapeColor
    }
    this.shape = null
  }

  clearLines() {
    let lines = 0
    for (let r = ROWS-1; r>=0; r--) {
      if (this.board[r].every(v => v !== 0)) {
        this.board.splice(r,1)
        this.board.unshift(Array(COLS).fill(0))
        lines++
        r++
      }
    }
    if (lines>0) {
      this.score += lines * 10
      this.events.emit('scoreChanged', this.score)
      this.totalLinesCleared += lines
      // for every 5 lines cleared, roll a reward
      while (this.totalLinesCleared >= 5) {
        this.totalLinesCleared -= 5
        const roll = Phaser.Math.Between(1, 100)
        // 1-30: none, 31-60: knife, 61-90: pistol, 91-100: bomb
        let item: string
        if (roll <= 30) item = 'none'
        else if (roll <= 60) item = 'knife'
        else if (roll <= 90) item = 'pistol'
        else item = 'bomb'
        this.events.emit('reward', item)
        if (item !== 'none') this.showReward(item)
      }
    }
  }

  showReward(item: string) {
    try {
      const centerX = (COLS * CELL) / 2
      const centerY = (ROWS * CELL) / 2
      // show smaller and slower reward animation
      const targetScale = 0.55
      const img = this.add.image(centerX, centerY, item).setScale(targetScale * 0.2).setDepth(100)
      img.setAlpha(0)
      this.tweens.add({
        targets: img,
        y: centerY - 40,
        scale: { from: targetScale * 0.2, to: targetScale },
        alpha: { from: 0, to: 1 },
        ease: 'Back.Out',
        duration: 700,
        onComplete: () => {
          this.tweens.add({
            targets: img,
            y: img.y - 30,
            alpha: { from: 1, to: 0 },
            duration: 900,
            onComplete: () => img.destroy()
          })
        }
      })
    } catch (e) {
      // ignore animation errors
    }
  }

  render() {
    if (!this.graphics) return
    this.graphics.clear()
    this.graphics.fillStyle(0x111111)
    this.graphics.fillRect(0, 0, this.scale.width, this.scale.height)

    for (let r=0;r<ROWS;r++) for (let c=0;c<COLS;c++) {
      const v = this.board[r][c]
      if (v) {
        this.graphics.fillStyle(v)
        this.graphics.fillRect(c*CELL, r*CELL, CELL-2, CELL-2)
      }
    }

    if (this.shape) {
      const n = this.shape.length
      this.graphics.fillStyle(this.shapeColor)
      for (let r=0;r<n;r++) for (let c=0;c<n;c++) if (this.shape[r][c]) {
        const x = (this.shapeX + c) * CELL
        const y = (this.shapeY + r) * CELL
        this.graphics.fillRect(x, y, CELL-2, CELL-2)
      }
    }

    // next piece preview — transparent overlay, top-left
    this.renderPreview()
  }

  private renderPreview() {
    if (!this.graphics) return
    const pw = 60
    const ph = 60
    const px = this.scale.width - pw - 6
    const py = 6

    // semi-transparent backdrop
    this.graphics.fillStyle(0x1a1a2e, 0.65)
    this.graphics.fillRect(px, py, pw, ph)
    this.graphics.lineStyle(1, 0x666688, 0.5)
    this.graphics.strokeRect(px, py, pw, ph)

    // reposition "NEXT" label above the panel
    if (this.nextLabel) {
      this.nextLabel.setPosition(px + pw / 2, py - 2)
    }

    // draw next shape centered in preview
    const ns = this.nextShape
    const n = ns.length
    const previewCell = 13
    const offsetX = px + (pw - n * previewCell) / 2
    const offsetY = py + (ph - n * previewCell) / 2
    this.graphics.fillStyle(COLORS[this.nextShapeIndex], 0.9)
    for (let r=0; r<n; r++) for (let c=0; c<n; c++) if (ns[r][c]) {
      this.graphics.fillRect(offsetX + c * previewCell, offsetY + r * previewCell, previewCell - 2, previewCell - 2)
    }
  }
}

export function createGame(container: HTMLElement) {
  const GAME_W = COLS * CELL
  const GAME_H = ROWS * CELL
  const config: Phaser.Types.Core.GameConfig = {
    type: Phaser.AUTO,
    parent: container,
    width: GAME_W,
    height: GAME_H,
    scale: {
      mode: Phaser.Scale.RESIZE,
      autoCenter: Phaser.Scale.CENTER_BOTH
    },
    backgroundColor: '#000000',
    scene: [TetrisScene]
  }

  const game = new Phaser.Game(config)

  // ensure the canvas fits the container and doesn't overflow layout
  setTimeout(() => {
    try {
      const anyGame: any = game
      const canvas: HTMLCanvasElement | undefined = anyGame.canvas || (anyGame.renderer && anyGame.renderer.canvas)
      if (canvas) {
        canvas.style.maxWidth = '100%'
        canvas.style.width = '100%'
        canvas.style.height = '100%'
        canvas.style.display = 'block'
        canvas.style.boxSizing = 'border-box'
        canvas.style.margin = '0'
      }
      // resize scale to current container size
      const w = Math.max(1, Math.floor(container.clientWidth))
      const h = Math.max(1, Math.floor(container.clientHeight))
      if (game.scale && game.scale.resize) game.scale.resize(w, h)
    } catch (e) {}
  }, 0)

  return game
}
