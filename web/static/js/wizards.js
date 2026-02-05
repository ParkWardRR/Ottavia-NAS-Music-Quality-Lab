// Ottavia Wizards - Interactive Onboarding & Help System

document.addEventListener('alpine:init', () => {
  // Wizard definitions
  const WIZARDS = {
    welcome: {
      id: 'welcome',
      title: 'Welcome to Ottavia',
      description: 'Your Music Quality Lab',
      steps: [
        {
          title: 'Welcome to Ottavia',
          content: `
            <div class="text-center mb-6">
              <div class="w-20 h-20 mx-auto rounded-2xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center shadow-lg shadow-blue-500/25 mb-4">
                <svg class="w-10 h-10 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3"></path>
                </svg>
              </div>
              <h3 class="text-2xl font-bold mb-2">Welcome to Ottavia</h3>
              <p class="text-gray-500 dark:text-gray-400">Your personal Music Quality Lab</p>
            </div>
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              Ottavia helps you analyze, organize, and improve your music library. Let's take a quick tour of the main features.
            </p>
            <div class="grid grid-cols-2 gap-3 text-sm">
              <div class="p-3 rounded-xl bg-blue-50 dark:bg-blue-500/10">
                <span class="font-medium text-blue-600 dark:text-blue-400">Audio Analysis</span>
                <p class="text-gray-500 dark:text-gray-400 text-xs mt-1">Detect quality issues</p>
              </div>
              <div class="p-3 rounded-xl bg-purple-50 dark:bg-purple-500/10">
                <span class="font-medium text-purple-600 dark:text-purple-400">Metadata</span>
                <p class="text-gray-500 dark:text-gray-400 text-xs mt-1">Fix tags & info</p>
              </div>
              <div class="p-3 rounded-xl bg-emerald-50 dark:bg-emerald-500/10">
                <span class="font-medium text-emerald-600 dark:text-emerald-400">Album Art</span>
                <p class="text-gray-500 dark:text-gray-400 text-xs mt-1">Find missing covers</p>
              </div>
              <div class="p-3 rounded-xl bg-amber-50 dark:bg-amber-500/10">
                <span class="font-medium text-amber-600 dark:text-amber-400">Conversions</span>
                <p class="text-gray-500 dark:text-gray-400 text-xs mt-1">Export to any format</p>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Adding a Library',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              First, you'll need to add your music library. Click the <strong>"Add Library"</strong> button on the dashboard to get started.
            </p>
            <div class="bg-gray-100 dark:bg-gray-800 rounded-xl p-4 mb-4">
              <div class="flex items-center gap-3 mb-3">
                <div class="w-8 h-8 rounded-lg bg-blue-500 flex items-center justify-center">
                  <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                  </svg>
                </div>
                <span class="font-medium">Add Library</span>
              </div>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                Point to any folder containing music files. Ottavia supports FLAC, MP3, AAC, ALAC, WAV, and more.
              </p>
            </div>
            <div class="text-sm text-amber-600 dark:text-amber-400 flex items-start gap-2">
              <svg class="w-5 h-5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              <span>Ottavia works with NAS drives, external drives, and local folders.</span>
            </div>
          `,
          highlight: '[data-wizard-target="add-library"]'
        },
        {
          title: 'Scanning Your Library',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              After adding a library, click <strong>"Scan"</strong> to analyze your music. The scanner will:
            </p>
            <ul class="space-y-3 mb-4">
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-emerald-100 dark:bg-emerald-500/20 flex items-center justify-center flex-shrink-0 mt-0.5">
                  <svg class="w-4 h-4 text-emerald-600 dark:text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                  </svg>
                </div>
                <span class="text-gray-600 dark:text-gray-300">Index all audio files and extract metadata</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-emerald-100 dark:bg-emerald-500/20 flex items-center justify-center flex-shrink-0 mt-0.5">
                  <svg class="w-4 h-4 text-emerald-600 dark:text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                  </svg>
                </div>
                <span class="text-gray-600 dark:text-gray-300">Detect audio format and quality</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-emerald-100 dark:bg-emerald-500/20 flex items-center justify-center flex-shrink-0 mt-0.5">
                  <svg class="w-4 h-4 text-emerald-600 dark:text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
                  </svg>
                </div>
                <span class="text-gray-600 dark:text-gray-300">Identify missing or low-quality album art</span>
              </li>
            </ul>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              Scans run in the background so you can keep using the app.
            </p>
          `,
          highlight: null
        },
        {
          title: 'Audio Scan Analysis',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              For deep analysis, run an <strong>Audio Scan</strong> on individual tracks. This reveals:
            </p>
            <div class="space-y-3 mb-4">
              <div class="flex items-center gap-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-500 flex items-center justify-center">
                  <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path>
                  </svg>
                </div>
                <div>
                  <span class="font-medium text-gray-900 dark:text-white">Spectrum Analysis</span>
                  <p class="text-xs text-gray-500 dark:text-gray-400">Detect lossy transcodes masquerading as lossless</p>
                </div>
              </div>
              <div class="flex items-center gap-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-amber-500 to-orange-500 flex items-center justify-center">
                  <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                  </svg>
                </div>
                <div>
                  <span class="font-medium text-gray-900 dark:text-white">Dynamic Range</span>
                  <p class="text-xs text-gray-500 dark:text-gray-400">Check for over-compression and clipping</p>
                </div>
              </div>
              <div class="flex items-center gap-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center">
                  <svg class="w-5 h-5 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.536 8.464a5 5 0 010 7.072m2.828-9.9a9 9 0 010 12.728M5.586 15.536a5 5 0 001.414 1.414m2.828-9.9a9 9 0 012.828-2.828"></path>
                  </svg>
                </div>
                <div>
                  <span class="font-medium text-gray-900 dark:text-white">Phase & Loudness</span>
                  <p class="text-xs text-gray-500 dark:text-gray-400">Stereo issues and LUFS measurements</p>
                </div>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'You\'re All Set!',
          content: `
            <div class="text-center mb-6">
              <div class="w-16 h-16 mx-auto rounded-full bg-emerald-100 dark:bg-emerald-500/20 flex items-center justify-center mb-4">
                <svg class="w-8 h-8 text-emerald-600 dark:text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
              </div>
              <h3 class="text-xl font-bold mb-2">You're Ready to Go!</h3>
            </div>
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              You can access more guides anytime from the <strong>Help</strong> menu in the sidebar.
            </p>
            <div class="bg-blue-50 dark:bg-blue-500/10 rounded-xl p-4">
              <p class="text-sm text-blue-700 dark:text-blue-300 font-medium mb-2">Quick Tips:</p>
              <ul class="text-sm text-blue-600 dark:text-blue-400 space-y-1">
                <li>• Click any track to view detailed analysis</li>
                <li>• Use the Issues page to find problems</li>
                <li>• Bulk edit metadata from the Albums view</li>
              </ul>
            </div>
          `,
          highlight: null
        }
      ]
    },

    library: {
      id: 'library',
      title: 'Library Setup',
      description: 'Learn how to add and manage music libraries',
      steps: [
        {
          title: 'Adding Your First Library',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              A library is a folder containing your music files. You can add multiple libraries for different collections.
            </p>
            <div class="space-y-3">
              <div class="p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <span class="font-medium">Name</span>
                <p class="text-sm text-gray-500 dark:text-gray-400">Give your library a descriptive name like "Main Collection" or "Hi-Res Audio"</p>
              </div>
              <div class="p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <span class="font-medium">Path</span>
                <p class="text-sm text-gray-500 dark:text-gray-400">The full path to your music folder, e.g., /mnt/music or /Users/you/Music</p>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Supported Formats',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              Ottavia supports all major audio formats:
            </p>
            <div class="grid grid-cols-2 gap-2 mb-4">
              <div class="p-2 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 text-center">
                <span class="font-mono text-emerald-600 dark:text-emerald-400">FLAC</span>
              </div>
              <div class="p-2 rounded-lg bg-emerald-50 dark:bg-emerald-500/10 text-center">
                <span class="font-mono text-emerald-600 dark:text-emerald-400">ALAC</span>
              </div>
              <div class="p-2 rounded-lg bg-blue-50 dark:bg-blue-500/10 text-center">
                <span class="font-mono text-blue-600 dark:text-blue-400">MP3</span>
              </div>
              <div class="p-2 rounded-lg bg-blue-50 dark:bg-blue-500/10 text-center">
                <span class="font-mono text-blue-600 dark:text-blue-400">AAC/M4A</span>
              </div>
              <div class="p-2 rounded-lg bg-purple-50 dark:bg-purple-500/10 text-center">
                <span class="font-mono text-purple-600 dark:text-purple-400">WAV</span>
              </div>
              <div class="p-2 rounded-lg bg-purple-50 dark:bg-purple-500/10 text-center">
                <span class="font-mono text-purple-600 dark:text-purple-400">AIFF</span>
              </div>
              <div class="p-2 rounded-lg bg-amber-50 dark:bg-amber-500/10 text-center">
                <span class="font-mono text-amber-600 dark:text-amber-400">OGG</span>
              </div>
              <div class="p-2 rounded-lg bg-amber-50 dark:bg-amber-500/10 text-center">
                <span class="font-mono text-amber-600 dark:text-amber-400">OPUS</span>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Scheduling Scans',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              You can set up automatic scans to keep your library updated:
            </p>
            <ul class="space-y-3">
              <li class="flex items-start gap-3">
                <div class="w-8 h-8 rounded-lg bg-blue-100 dark:bg-blue-500/20 flex items-center justify-center flex-shrink-0">
                  <span class="text-blue-600 dark:text-blue-400 font-bold text-sm">1</span>
                </div>
                <div>
                  <span class="font-medium">Manual Scan</span>
                  <p class="text-sm text-gray-500 dark:text-gray-400">Click "Scan" anytime to check for new files</p>
                </div>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-8 h-8 rounded-lg bg-blue-100 dark:bg-blue-500/20 flex items-center justify-center flex-shrink-0">
                  <span class="text-blue-600 dark:text-blue-400 font-bold text-sm">2</span>
                </div>
                <div>
                  <span class="font-medium">Scheduled Scan</span>
                  <p class="text-sm text-gray-500 dark:text-gray-400">Set daily/weekly scans in Settings</p>
                </div>
              </li>
            </ul>
          `,
          highlight: null
        }
      ]
    },

    audioscan: {
      id: 'audioscan',
      title: 'Audio Scan Guide',
      description: 'Understanding audio quality analysis',
      steps: [
        {
          title: 'What is Audio Scan?',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              Audio Scan performs deep analysis of your audio files using professional-grade algorithms to detect quality issues.
            </p>
            <div class="bg-gradient-to-r from-blue-500/10 to-purple-500/10 rounded-xl p-4 border border-blue-200 dark:border-blue-800">
              <p class="text-sm text-gray-600 dark:text-gray-300">
                <strong>Pro Tip:</strong> Audio Scan can detect "fake lossless" files - lossy MP3s that were re-encoded as FLAC, wasting disk space without any quality benefit.
              </p>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Spectrum Analysis',
          content: `
            <div class="mb-4">
              <div class="h-24 bg-gradient-to-r from-blue-500 via-cyan-500 to-purple-500 rounded-xl opacity-20 mb-2"></div>
              <p class="text-xs text-center text-gray-500 dark:text-gray-400">Example: Frequency spectrum visualization</p>
            </div>
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              The spectrum chart shows frequency content over time:
            </p>
            <ul class="space-y-2 text-sm">
              <li class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-emerald-500"></span>
                <span><strong>Full bandwidth</strong> = True lossless quality</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-amber-500"></span>
                <span><strong>Cutoff at 16kHz</strong> = Likely MP3 source</span>
              </li>
              <li class="flex items-center gap-2">
                <span class="w-3 h-3 rounded-full bg-red-500"></span>
                <span><strong>Cutoff below 16kHz</strong> = Low bitrate source</span>
              </li>
            </ul>
          `,
          highlight: null
        },
        {
          title: 'Dynamic Range',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              Dynamic range measures the difference between quiet and loud parts:
            </p>
            <div class="space-y-3 mb-4">
              <div class="flex items-center gap-3">
                <div class="w-12 h-8 rounded bg-emerald-500 flex items-center justify-center text-white text-xs font-bold">DR14+</div>
                <span class="text-sm text-gray-600 dark:text-gray-300">Excellent - Full dynamic range preserved</span>
              </div>
              <div class="flex items-center gap-3">
                <div class="w-12 h-8 rounded bg-blue-500 flex items-center justify-center text-white text-xs font-bold">DR10-13</div>
                <span class="text-sm text-gray-600 dark:text-gray-300">Good - Typical for most modern music</span>
              </div>
              <div class="flex items-center gap-3">
                <div class="w-12 h-8 rounded bg-amber-500 flex items-center justify-center text-white text-xs font-bold">DR6-9</div>
                <span class="text-sm text-gray-600 dark:text-gray-300">Compressed - "Loudness war" victim</span>
              </div>
              <div class="flex items-center gap-3">
                <div class="w-12 h-8 rounded bg-red-500 flex items-center justify-center text-white text-xs font-bold">DR&lt;6</div>
                <span class="text-sm text-gray-600 dark:text-gray-300">Over-compressed - May have clipping</span>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Running Audio Scan',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              To run Audio Scan on a track:
            </p>
            <ol class="space-y-3 mb-4">
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">1</div>
                <span class="text-gray-600 dark:text-gray-300">Navigate to any track detail page</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">2</div>
                <span class="text-gray-600 dark:text-gray-300">Click the "Run Audio Scan" button</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">3</div>
                <span class="text-gray-600 dark:text-gray-300">View results in the interactive charts</span>
              </li>
            </ol>
            <p class="text-sm text-gray-500 dark:text-gray-400">
              You can also run bulk Audio Scans from the Albums or Tracks pages.
            </p>
          `,
          highlight: null
        }
      ]
    },

    metadata: {
      id: 'metadata',
      title: 'Metadata Editing',
      description: 'Learn how to fix and improve track tags',
      steps: [
        {
          title: 'Understanding Metadata',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              Metadata is the information stored in your audio files - artist, album, track number, and more.
            </p>
            <div class="grid grid-cols-2 gap-2">
              <div class="p-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm">
                <span class="text-gray-500 dark:text-gray-400">Title</span>
                <p class="font-medium truncate">Song Name</p>
              </div>
              <div class="p-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm">
                <span class="text-gray-500 dark:text-gray-400">Artist</span>
                <p class="font-medium truncate">Artist Name</p>
              </div>
              <div class="p-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm">
                <span class="text-gray-500 dark:text-gray-400">Album</span>
                <p class="font-medium truncate">Album Title</p>
              </div>
              <div class="p-2 rounded-lg bg-gray-100 dark:bg-gray-800 text-sm">
                <span class="text-gray-500 dark:text-gray-400">Year</span>
                <p class="font-medium">2024</p>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Editing Single Tracks',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              To edit a single track's metadata:
            </p>
            <ol class="space-y-3">
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">1</div>
                <span class="text-gray-600 dark:text-gray-300">Click on any track to open its detail page</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">2</div>
                <span class="text-gray-600 dark:text-gray-300">Click "Edit Metadata" to modify tags</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">3</div>
                <span class="text-gray-600 dark:text-gray-300">Preview changes before saving</span>
              </li>
              <li class="flex items-start gap-3">
                <div class="w-6 h-6 rounded-full bg-blue-500 text-white flex items-center justify-center text-sm font-bold flex-shrink-0">4</div>
                <span class="text-gray-600 dark:text-gray-300">Click "Apply" to write changes to the file</span>
              </li>
            </ol>
          `,
          highlight: null
        },
        {
          title: 'Bulk Editing',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              Edit multiple tracks at once from the Album view:
            </p>
            <div class="space-y-3 mb-4">
              <div class="p-3 rounded-xl bg-blue-50 dark:bg-blue-500/10">
                <span class="font-medium text-blue-700 dark:text-blue-300">Normalize Album Artist</span>
                <p class="text-sm text-blue-600 dark:text-blue-400">Set consistent album artist across all tracks</p>
              </div>
              <div class="p-3 rounded-xl bg-purple-50 dark:bg-purple-500/10">
                <span class="font-medium text-purple-700 dark:text-purple-300">Fix Track Numbering</span>
                <p class="text-sm text-purple-600 dark:text-purple-400">Auto-number tracks in the correct order</p>
              </div>
              <div class="p-3 rounded-xl bg-emerald-50 dark:bg-emerald-500/10">
                <span class="font-medium text-emerald-700 dark:text-emerald-300">Set Year/Genre</span>
                <p class="text-sm text-emerald-600 dark:text-emerald-400">Apply the same year or genre to all tracks</p>
              </div>
            </div>
          `,
          highlight: null
        }
      ]
    },

    artwork: {
      id: 'artwork',
      title: 'Album Art Guide',
      description: 'Find and manage album artwork',
      steps: [
        {
          title: 'Finding Missing Artwork',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              The Album Art page shows all albums missing cover art or with low-resolution images.
            </p>
            <div class="flex items-center gap-4 p-4 rounded-xl bg-gray-100 dark:bg-gray-800 mb-4">
              <div class="w-16 h-16 rounded-lg bg-gray-300 dark:bg-gray-700 flex items-center justify-center">
                <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"></path>
                </svg>
              </div>
              <div>
                <p class="font-medium">Missing Album Art</p>
                <p class="text-sm text-gray-500 dark:text-gray-400">No embedded artwork found</p>
              </div>
            </div>
          `,
          highlight: null
        },
        {
          title: 'Adding Artwork',
          content: `
            <p class="text-gray-600 dark:text-gray-300 mb-4">
              You can add artwork in several ways:
            </p>
            <div class="space-y-3">
              <div class="flex items-start gap-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <div class="w-8 h-8 rounded-lg bg-blue-500 flex items-center justify-center flex-shrink-0">
                  <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path>
                  </svg>
                </div>
                <div>
                  <span class="font-medium">Upload Image</span>
                  <p class="text-sm text-gray-500 dark:text-gray-400">Upload a JPG or PNG file from your computer</p>
                </div>
              </div>
              <div class="flex items-start gap-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <div class="w-8 h-8 rounded-lg bg-purple-500 flex items-center justify-center flex-shrink-0">
                  <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
                  </svg>
                </div>
                <div>
                  <span class="font-medium">Search Online</span>
                  <p class="text-sm text-gray-500 dark:text-gray-400">Fetch from Cover Art Archive / MusicBrainz</p>
                </div>
              </div>
              <div class="flex items-start gap-3 p-3 rounded-xl bg-gray-100 dark:bg-gray-800">
                <div class="w-8 h-8 rounded-lg bg-emerald-500 flex items-center justify-center flex-shrink-0">
                  <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"></path>
                  </svg>
                </div>
                <div>
                  <span class="font-medium">Extract from Folder</span>
                  <p class="text-sm text-gray-500 dark:text-gray-400">Use cover.jpg/folder.jpg from album folder</p>
                </div>
              </div>
            </div>
          `,
          highlight: null
        }
      ]
    }
  };

  // Wizard store
  Alpine.store('wizard', {
    active: null,
    currentStep: 0,
    completed: JSON.parse(localStorage.getItem('wizardsCompleted') || '[]'),

    get currentWizard() {
      return this.active ? WIZARDS[this.active] : null;
    },

    get step() {
      return this.currentWizard?.steps[this.currentStep] || null;
    },

    get totalSteps() {
      return this.currentWizard?.steps.length || 0;
    },

    get isFirstStep() {
      return this.currentStep === 0;
    },

    get isLastStep() {
      return this.currentStep === this.totalSteps - 1;
    },

    get progress() {
      if (!this.totalSteps) return 0;
      return ((this.currentStep + 1) / this.totalSteps) * 100;
    },

    start(wizardId) {
      if (WIZARDS[wizardId]) {
        this.active = wizardId;
        this.currentStep = 0;
        document.body.style.overflow = 'hidden';
      }
    },

    next() {
      if (this.currentStep < this.totalSteps - 1) {
        this.currentStep++;
        this.highlightElement();
      } else {
        this.complete();
      }
    },

    prev() {
      if (this.currentStep > 0) {
        this.currentStep--;
        this.highlightElement();
      }
    },

    goTo(step) {
      if (step >= 0 && step < this.totalSteps) {
        this.currentStep = step;
        this.highlightElement();
      }
    },

    complete() {
      if (this.active && !this.completed.includes(this.active)) {
        this.completed.push(this.active);
        localStorage.setItem('wizardsCompleted', JSON.stringify(this.completed));
      }
      this.close();
    },

    close() {
      this.removeHighlight();
      this.active = null;
      this.currentStep = 0;
      document.body.style.overflow = '';
    },

    skip() {
      this.close();
    },

    highlightElement() {
      this.removeHighlight();
      const selector = this.step?.highlight;
      if (selector) {
        const el = document.querySelector(selector);
        if (el) {
          el.classList.add('wizard-highlight');
          el.scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      }
    },

    removeHighlight() {
      document.querySelectorAll('.wizard-highlight').forEach(el => {
        el.classList.remove('wizard-highlight');
      });
    },

    isCompleted(wizardId) {
      return this.completed.includes(wizardId);
    },

    reset() {
      this.completed = [];
      localStorage.removeItem('wizardsCompleted');
    },

    // Check if should show welcome wizard
    shouldShowWelcome() {
      return !this.isCompleted('welcome') && !localStorage.getItem('welcomeWizardDismissed');
    },

    dismissWelcome() {
      localStorage.setItem('welcomeWizardDismissed', 'true');
    },

    // Get available wizards
    getAvailableWizards() {
      return Object.values(WIZARDS).map(w => ({
        id: w.id,
        title: w.title,
        description: w.description,
        completed: this.isCompleted(w.id)
      }));
    }
  });
});

// Auto-show welcome wizard for new users
document.addEventListener('DOMContentLoaded', () => {
  setTimeout(() => {
    if (Alpine.store('wizard').shouldShowWelcome()) {
      Alpine.store('wizard').start('welcome');
    }
  }, 1000);
});
