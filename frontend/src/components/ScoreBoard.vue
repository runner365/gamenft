<script setup lang="ts">
import { defineProps } from 'vue'

const props = defineProps<{ score: number; knife: number; pistol: number; bomb: number }>()
const emit = defineEmits<{
  (e: 'use', item: string): void
  (e: 'sell', item: string): void
}>()

import knifeImg from '../../imgs/knife.png'
import pistolImg from '../../imgs/pistol.png'
import bombImg from '../../imgs/bomb.png'

function useItem(item: string) {
  if (item === 'knife' && props.knife > 0) emit('use', 'knife')
  if (item === 'pistol' && props.pistol > 0) emit('use', 'pistol')
  if (item === 'bomb' && props.bomb > 0) emit('use', 'bomb')
}

function sellItem(item: string) {
  if (item === 'knife' && props.knife > 0) emit('sell', 'knife')
  if (item === 'pistol' && props.pistol > 0) emit('sell', 'pistol')
  if (item === 'bomb' && props.bomb > 0) emit('sell', 'bomb')
}
</script>

<template>
  <div class="scoreboard">
    <h3>Score</h3>
    <div class="value">{{ props.score }}</div>
    
    <div class="rewards">
      <div class="reward-row">
        <img :src="knifeImg" alt="knife" />
        <span> x {{ props.knife }}</span>
        <button class="use-btn" @click="useItem('knife')" :disabled="props.knife <= 0">Use</button>
        <button class="sell-btn" @click="sellItem('knife')" :disabled="props.knife <= 0">Sell</button>
      </div>
      <div class="reward-row">
        <img :src="pistolImg" alt="pistol" />
        <span> x {{ props.pistol }}</span>
        <button class="use-btn" @click="useItem('pistol')" :disabled="props.pistol <= 0">Use</button>
        <button class="sell-btn" @click="sellItem('pistol')" :disabled="props.pistol <= 0">Sell</button>
      </div>
      <div class="reward-row">
        <img :src="bombImg" alt="bomb" />
        <span> x {{ props.bomb }}</span>
        <button class="use-btn" @click="useItem('bomb')" :disabled="props.bomb <= 0">Use</button>
        <button class="sell-btn" @click="sellItem('bomb')" :disabled="props.bomb <= 0">Sell</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.scoreboard {
  width: 120px;
  padding: 6px;
  background: #222;
  color: #fff;
  border-radius: 6px;
}
.scoreboard h3 {
  margin: 0 0 6px 0;
  font-size: 14px;
}
.scoreboard .value {
  font-size: 20px;
  font-weight: 600;
}
.rewards { margin-top: 6px; display:flex;flex-direction:column;gap:4px }
.reward-row { display:flex;align-items:center;gap:4px }
.reward-row img { width:24px;height:24px }
.use-btn { padding:4px 6px; font-size:11px }
.sell-btn { padding:4px 6px; font-size:11px; background:#4a7c2e; border-color:#4a7c2e; color:#fff }
</style>
