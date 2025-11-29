import React from 'react'
import {createRoot} from 'react-dom/client'
import App from './App'
import { ThemeProvider } from './components/ThemeProvider'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <ThemeProvider defaultTheme="system" storageKey="quicksearch-theme">
            <App/>
        </ThemeProvider>
    </React.StrictMode>
)
