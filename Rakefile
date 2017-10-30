PKG = 'github.com/cleafy/promqueen'.freeze

task :ensure do
	ln_sf ENV['PWD'], "#{ENV['GOPATH']}/src/#{PKG}"
	sh "cd #{ENV['GOPATH']}/src/#{PKG} && dep ensure"
	cd ENV['PWD']
	Dir.chdir('vendor') do
		Dir['*/*/*'].each do |d|
			puts "Linking #{d} into #{ENV['GOPATH']}"
			rm_rf "#{ENV['GOPATH']}/src/#{d}"
			mkdir_p "#{ENV['GOPATH']}/src/#{File.dirname(d)}"
			ln_sf "#{Dir.pwd}/#{d}", "#{ENV['GOPATH']}/src/#{d}"
		end
	end
end
