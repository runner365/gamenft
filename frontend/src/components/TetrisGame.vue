<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { createGame } from '../game/tetris'

const emit = defineEmits<{
  (e: 'score', value: number): void
  (e: 'reward', item: string): void
}>()

const container = ref<HTMLElement | null>(null)
let game: any = null
const running = ref(false)

onMounted(() => {
  if (container.value) {
    game = createGame(container.value)
    console.log('TetrisGame: created game instance', game)
    // when the scene is available, forward score and reward events to parent
      setTimeout(() => {
        try {
          console.log('TetrisGame: container size', container.value?.clientWidth, 'x', container.value?.clientHeight)
          const anyGame: any = game
          const canvas: HTMLCanvasElement | undefined = anyGame.canvas || (anyGame.renderer && anyGame.renderer.canvas)
          if (canvas) console.log('TetrisGame: canvas size', canvas.width, 'x', canvas.height, 'style', canvas.style.width, canvas.style.height)
        } catch (e) {}
      }, 200)
    const tryAttach = () => {
      try {
        const scene = game.scene && game.scene.getScene && game.scene.getScene('TetrisScene')
        if (scene && scene.events) {
          scene.events.on('scoreChanged', (s: number) => emit('score', s))
          scene.events.on('reward', (it: string) => emit('reward', it))
          // emit initial score
          emit('score', 0)
          console.log('TetrisGame: attached to scene', scene)
          return
        }
      } catch (e) {}
      setTimeout(tryAttach, 100)
    }
    tryAttach()
  }
})

onBeforeUnmount(() => {
  if (game) game.destroy(true)
})

function start() {
  const scene = game && game.scene && game.scene.getScene && game.scene.getScene('TetrisScene')
  if (scene && scene.startGame) { scene.startGame(); running.value = true }
}

function pause() {
  const scene = game && game.scene && game.scene.getScene && game.scene.getScene('TetrisScene')
  if (scene && scene.pauseGame) { scene.pauseGame(); running.value = false }
}

function useItem(item: string) {
  const scene = game && game.scene && game.scene.getScene && game.scene.getScene('TetrisScene')
  if (scene && scene.events) scene.events.emit('useItem', item)
}

defineExpose({ start, pause, useItem })
</script>

<template>
  <div class="tetris-wrapper">
    <div class="tetris-container" ref="container" />
  </div>
</template>

<style scoped>
.tetris-wrapper { flex: 1; display: flex; justify-content: center; align-items: flex-start; }
.tetris-container {
  width: 100%;
  max-width: 300px;
  min-height: 500px;
  max-height: 85vh;
  overflow: hidden;
  background: #fff;
  border: 1px solid rgba(0,0,0,0.12);
}
</style>
